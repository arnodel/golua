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

### What a copy shares

A copy has the values of the registers.  A local variable that lives in a
register is therefore as it was at the capture, each time the continuation is
called.  Tables are shared, and so are the variables that a function captures
and assigns, which live in boxes that the registers point to.  A Scheme
frontend, where every assigned variable must be shared, has to box them all.

## A type, not a flag

A frozen frame is a `*frozenCont`, which is a `LuaCont` under another type
with the same memory.  The alternative was a flag in `LuaCont`.

- Each place that pushes into a continuation already tests whether it is a
  `*LuaCont`, to take a fast path.  A `*frozenCont` fails that test and ends
  up on the slow path, where the copy is made.  A flag would add a load and
  a branch to each of those places, for every program.  A flag test that was
  never taken, on one family of opcodes, was measured at +1.45% before.
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
Go function that returns the continuation to run next.  A Go function that
needs the results of Lua code before it can go on (`pcall`, a metamethod
call, the comparison function of `table.sort`, the body of a coroutine)
starts a loop of its own and waits for it.  So the loops of a thread are
nested as the Go calls that started them are.

    Go stack of a thread                 frames (each returns to the next)

    RunContinuation  loop 1  -- runs -->  main chunk ... Termination 1
      pcall (waiting)
        RunContinuation  loop 2  -- runs -->  g, h ... Termination 2

A continuation captured in `h` holds `h`, `g` and Termination 2: "carry on in
`h`, and when `g` returns, give `pcall` its results".  The Go frame of
`pcall` is not part of it and cannot be copied.  So that continuation can
carry on in loop 2 and nowhere else, and a `ContFunction` keeps the loop it
was captured in (a `runLoop`, which a thread makes for a loop when a
continuation is first captured in it).

## Calling a continuation from another loop

Loops are nested, so when a continuation is called in a loop that is not its
own, in its own thread, there are two cases.

- Its loop has ended: the Go code that waited for its frames has returned.
  There is nowhere to carry on, and the call is an error.  A `runLoop` has a
  flag for this, set when the loop ends.
- Its loop is still there, further down the Go stack, waiting for Go code
  that has called back into Lua.  The continuation can carry on once that Go
  code has returned, so the call makes it return: it travels as an error
  value (`contFunctionCall`) which Go code passes on as it passes errors on.
  Each `RunContinuation` on the way ends, closing the to-be-closed variables
  of the frames it leaves, and the loop of the capture takes the call up and
  runs the continuation.

So a continuation is a way out of a `pcall`, at the cost of an error that is
caught (about 650 ns for either), and not a way back into one.

A Go function that turns the errors of the Lua code it calls into values has
to let this one through, which `rt.IsContFunctionCall` is for.  `pcall`,
`xpcall` and the reader function of `load` do.  A function that does not gets
a message and the continuation is not called.  `runtime.callcontext` is the
exception by design: a context is not left by calling a continuation
captured outside it, and ends with an error instead, because the code that
starts a context gets it back whatever runs in it.

Calling a continuation in another thread is an error: its loop is one of
that thread's.

## The first version, and what this one costs

The first version copied every frame down to the bottom at the capture and
again at each call.  That was about 180 ns per frame per copy, nearly all of
it allocation: 17.5 µs to capture under 105 frames.

With frozen frames a capture is a pointer conversion and an allocation, and a
capture followed by a call is about 185 ns at any depth (M1 Max, without
quotas).  A generator made of two continuations gives a value in about
815 ns, where a coroutine takes about 600 ns.  Programs that do not capture
continuations run the same code as before, apart from the order of two type
tests at a return.
