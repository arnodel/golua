# Implementation notes for first-class continuations

Golua's runtime has no call stack.  A running function is a continuation (a
`Cont`), a call makes the callee's continuation with the caller's as the one
to return to, and a return pushes values into that one and runs it.  This
note is about making such a continuation a value that a program can keep and
call any number of times, which is what `callcc` in the `cont` library gives
Lua programs and what `call/cc` needs in a Scheme frontend.

The code is in `runtime/contfunction.go` and `runtime/frozencont.go`.

## What is exported

- `Thread.CaptureCont(next)` returns a function `k` that stands for the
  continuation `next`, and the continuation to use in place of `next` from
  then on.
- `ContFunction` is the type of `k`.  To Lua it is a function: calling it
  makes `next` carry on with the arguments of the call.

`callcc` is library code on top of this (`lib/contlib`): it captures the
continuation of its own call and calls its argument with it.

## A frame runs once

A `LuaCont` (a frame) is the registers of a Lua function and its place in the
code.  Running it changes both, and when it returns it goes back to a pool to
be the frame of another call.  So a frame can be run from a given state once,
and calling a continuation a second time needs that state again.

## Frozen frames, copied when pushed into

A captured frame is *frozen*: it never runs again and nothing is pushed into
it.  What runs is a copy, and the copy is made at the last moment, when
something is about to push values into the frozen frame:

- the run loop, at a return or when it pushes arguments or results into the
  continuation in a register (`contToPush`);
- a Go function about to run, whose results go to its next continuation
  (`GoCont.RunInThread`);
- the call of a `ContFunction` (`reinstate`).

The copy (`frozenCont.thaw`) has the same registers, except that it refers to
the frame it returns to as frozen.  So only the frame at the capture is
marked at the capture, whatever the depth of the calls, and the frames below
it are copied one at a time, each when a copy returns into it.  A frame that
is never returned into again is never copied.

A frame can also hold frames it has made for calls whose arguments it is
still computing (`f(x, g())` makes the frame of `f` before it calls `g`).
They are copied with it.

### Protected calls

The function that `pcall` calls does not return to the frame that called
`pcall` but to a `protectedCall`, a continuation that stays between the two
and marks where an error stops (`runtime/protectedcall.go`).  It is not a
frame, but like a frame it is changed by being returned into: it keeps the
results until the variables of the function are closed.  So it is copied as
a frame is.  A copy of a frame that returns to a protected call returns to a
copy of that (`protectedCall.clone`), which in turn returns to what the
original did, frozen.

The first idea was to freeze what a protected call returns to and leave the
call itself shared between the copies of the frame, since it holds results
only from the return of the function to its own run.  But closing a variable
comes in between, and a `__close` handler can call a continuation there:
the results of the return that was left would be given with those of the
next one.

The thread counts the protected calls it is in, to limit their depth.  A
`ContFunction` keeps the count at its capture, which is the count again when
it is called.

### What a copy shares

A copy has the values of the registers.  A local variable that lives in a
register is therefore as it was at the capture, each time the continuation is
called.  Tables are shared, and so are the variables that a function captures
and assigns, which live in boxes that the registers point to.  A Scheme
frontend, where every assigned variable must be shared, has to box them all.

## A type, not a flag

A frozen frame is a `*frozenCont`, which is a `LuaCont` under another type
with the same memory.  The alternative was a flag in `LuaCont`.

- The fast paths that the run loop has for a `*LuaCont` (a call that stores
  its arguments into the callee's registers, for one) do not take a
  `*frozenCont`, with no test added to them.  Where the loop reads a
  continuation to push into it or to return to it, the test for a frozen
  frame is on the type of the interface value it has just read: a
  comparison.  A flag would be a load from the frame and a branch at each
  of those places.  A flag test that was never taken, on one family of
  opcodes, was measured at +1.45% before.
- `frozenCont.Push` panics, so pushing into a frozen frame by mistake is
  found at once.
- The cost is that the code which tells the kinds of continuation apart must
  know this one (`AsCont`, `TryCont`, `isLuaFrame`), and that being frozen
  belongs to the reference and not to the frame.  It holds because the
  frozen reference replaces the only way to the frame: `CaptureCont` returns
  it for the caller to use, and a copy has it in register 0.

## Loops: where a continuation can carry on

`Thread.RunContinuation` is a loop that runs continuations one after the
other.  A Lua call or return is one turn of the loop, and so is the call of a
Go function that returns the continuation to run next, as `pcall` does.  A
Go function that needs the results of Lua code before it can go on (a
metamethod call, the comparison function of `table.sort`) starts a loop of
its own and waits for it.  So the loops of a thread are nested as the Go
calls that started them are.

    Go stack of a thread                 frames (each returns to the next)

    RunContinuation  loop 1  -- runs -->  main chunk ... Termination 1
      table.sort (waiting)
        RunContinuation  loop 2  -- runs -->  g, h ... Termination 2

A continuation captured in `h` holds `h`, `g` and Termination 2: "carry on in
`h`, and when `g` returns, give `table.sort` its result".  The Go frame of
`table.sort` is not part of it and cannot be copied.  So that continuation
can carry on in loop 2 and nowhere else, and a `ContFunction` keeps the loop
it was captured in (a `runLoop`, which a thread makes for a loop when a
continuation is first captured in it).  Calling it in another loop is an
error, as is calling it in another thread.

A coroutine has a loop of its own, which is the first of its thread.  While
the coroutine has no goroutine (`runtime/lightthread.go`) that loop returns
at each yield and starts again at the next resume, and if the coroutine is
given a goroutine it goes on there.  It is one loop all along for a
continuation: the frames are the same, and nothing waits for them on a Go
stack.  So the thread keeps its `runLoop` through all of this.

`callcc` and the Go function that calls a `ContFunction` are declared
`NoYieldAcross`: they hand over to a continuation and wait for nothing.
Without the declaration a coroutine would be given a goroutine to call them.

## A call of a continuation is decided when it runs

To Lua a `ContFunction` is a function, and it is called as one: the call is
that of a Go function (`ContFunction.call`) which finds out when it runs
whether the thread is in the loop of the continuation, and if so returns a
copy of the captured frame with the arguments pushed into it as the
continuation to run next.

An earlier version did this sooner for a call made in the continuation's own
loop: the code that makes the callee's continuation for a call (`Continue`)
returned the copy of the captured frame directly, and the arguments were
pushed into that.  It saved the Go call, 15 ns of about 140 on the call of a
continuation, and more when the call is a tail call.  It was wrong in two
ways, because the callee's continuation is made before the call happens.

- The arguments are computed after it, in the computation that is about to
  be left: in `k(f(x))` the frame of `f` runs between the two.  Anything
  the continuation changes in the thread when it is reinstated was changed
  for `f` too.
- Go code can ask for the continuation of a call in one loop and run it in
  a loop of its own.  The frames of the outer loop then ran in the inner
  one, and again when it had returned.

`pcall(k)` showed the first: the protected call was counted as under way
after the thread had gone back to the depth of the capture of `k`, and a
thousand of those were a "stack overflow".

## The first version, and what this one costs

The first version copied every frame down to the bottom at the capture and
again at each call.  That was about 180 ns per frame per copy, nearly all of
it allocation: 17.5 µs to capture under 105 frames.

With frozen frames a capture is a pointer conversion and an allocation,
about 180 ns at any depth, and a capture followed by a call is about 300 ns
(M1 Max, without quotas).  A generator made of two continuations gives a
value in about 940 ns, where a coroutine takes about 100 ns since it needs
no goroutine (it took about 600 ns before).

Programs that do not capture continuations run the code they ran before,
with one test more in two places: where the run loop reads a continuation to
push into it or to return to it, and on entry to a Go function, whose next
continuation may be frozen.  The second is about 0.6 ns for each call of a
Go function, which is 2 ns on a coroutine switch.

## Writing it so that the build stays as fast

Two things about the build decided how this is written, more than the
design did.  Both were measured against `lua5.5` at dee2d0d.

- **The profile counts lines.**  The profile the build is optimised with
  (`default.pgo`, see `benchmarks/README.md`) finds a call by its line in
  its function.  Lines added before the hot calls of `GoCont.RunInThread`
  and of the loop function `runContinuation` made a coroutine switch 5 ns
  slower with the profile, and no slower without it.  So what is added
  there is on lines that were there: a call on an empty line (`thawNext`),
  and two helpers in the place of the lines that counted the loops
  (`enterLoop`, `leaveLoop`).  Keeping the loop's record in
  `RunContinuation`, around the loop function, was tried too: it costs a
  call and a defer, 10 ns on each metamethod call.
- **The run loop is compiled as a whole.**  `contToPush` first read the
  register and thawed, as one call from the run loop, where the loop used
  to have the read inlined.  Programs that never push into a continuation
  were 3 to 5% slower for it without the profile (sieve, spectral-norm,
  md5), and back to their times with the run loop's source put back.  Now
  the loop reads the continuation as it did and `contToPush`, which is
  inlined, tests what it read.
