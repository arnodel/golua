# The call protocol: design note

Status: 8 November 2026.  Step 1 is done (#210).  Since then the runtime's
side of a call has got cheaper (#224 to #226, and the pushes, #229), and
this note was revised in that light: step 2 is now last and may not be
worth doing, and step 3 may take the trampoline out of Lua-to-Lua calls.

## Where this comes from

Counting the opcodes the benchmarks execute (see [ir-passes.md](ir-passes.md))
put the call protocol at 30 to 45% of all opcodes in the object-oriented
programs, at 17a888f (31 October).  A call `o:m(a, b)` compiled to

    r4 <- r2[K ("m")]
    r4 <- cont(r4)          -- make the callee's continuation
    push r4, r2             -- self
    push r4, r5
    push r4, r6
    call r4

and the callee began with `recv r1; recv r2; recv r3`.  Eight opcodes of
protocol for one call, and the measurements said that on these programs an
opcode costs about the same whatever it does: the trip through the run loop
is the price.

## The model

golua is a continuation machine: there is no stack, a frame is a
continuation, and a return is a call of the caller's continuation, which
every frame holds in `r0`.  Values move between frames by being *pushed* into
a continuation; the receiving frame says where they land.  It says so with
`recv` opcodes at the point where it resumes: at pc 0 for its parameters,
after a `call` for the call's results.  A push into a `LuaCont` reads the
`recv` at the continuation's pc, writes the value there and advances the pc;
when the frame runs, any `recv` still ahead of the pc sets its register to nil.
That is why `recv` never shows in the dynamic counts: it only executes for
values that were not supplied.

The redesign keeps the model and changes only how the two sides are written
down, and how cheaply the runtime moves between them.

## What a call costs now

On lua5.5 at 6e2ae23, `-tags noquotas`, an M1 Max: a call `f()` of an empty
Lua function costs about 46 ns over the loop around it, against 10.6 ns in C
Lua 5.5, and each argument about 5 ns more, against 1.5 ns.  A call
goes through these steps:

1. `r <- cont(rF)` makes the callee's frame: `NewLuaCont` takes registers,
   cells and the `LuaCont` from the thread's pools.
2. `call r (args)` pushes the listed values into it (see Pushing values).
3. The call saves the pc and returns the callee from `RunInThread` to the
   trampoline, `Thread.RunContinuation`.
4. The trampoline checks for pending finalizers (one load), records the
   current continuation and calls the callee's `RunInThread` through the
   `Cont` interface.
5. The callee's run loop starts: the function prologue, and reloading the
   pc, the code, the constants, the registers and the cells.
6. The callee returns with `tailcall r0 (results)`, which pushes the results
   into the caller, releases the callee's frame and returns the caller to the
   trampoline; steps 3 to 5 again, back into the caller.

Executed opcodes on the same revision, in percent (the receives are
counted where `Push` performs them, one per value received, so the shares are
of opcodes and received values together; only `cont`, `call` and `push` go
through the run loop):

| | cont | call | push | recv | protocol | jumpif | field access |
|---|---|---|---|---|---|---|---|
| regex-dna | 13.2 | 13.3 | 1.0 | 18.7 | 46.2 | 12.7 | 8.4 |
| towers | 8.2 | 14.8 | 0.0 | 21.3 | 44.3 | 4.9 | 26.2 |
| havlak | 8.4 | 14.6 | 0.9 | 19.6 | 43.5 | 7.5 | 23.7 |
| md5 | 5.4 | 10.1 | 0.0 | 26.4 | 41.8 | 1.2 | 0.0 |
| deltablue | 8.3 | 16.0 | 0.2 | 13.4 | 37.7 | 8.5 | 29.0 |
| queens | 5.4 | 10.8 | 0.0 | 21.1 | 37.3 | 14.7 | 15.3 |
| richards | 7.0 | 11.5 | 0.0 | 13.9 | 32.5 | 13.2 | 34.2 |
| bounce | 3.6 | 6.4 | 0.0 | 6.6 | 16.7 | 14.7 | 30.0 |
| mandelbrot, sieve, fannkuch, life | 0 | 0 | 0 | 0 | 0 | 11 to 22 | 0 to 2 |
| **mean of 23** | **5.5** | **8.9** | **0.2** | **12.4** | **27.0** | **11.2** | **14.5** |

Before step 1, with hoisting in, the protocol was 32% on the same count, a
third of it pushes; the pushes left are those of a trailing vararg or call
result.  Per call, the opcodes are few now (a `cont` and a `call`, and the
`tailcall` of the return); what remains costly is the runtime's work around
them, steps 1 and 3 to 6.

## Step 1, done: arguments on the call (#210)

`call rCont (args)`: `OpCall` (Type 5) holds the argument count in its 16
unused bits, and `ceil(n/3)` operand words follow, listing the argument
registers three per word in the A, B, C positions with their kind bits.  The
remaining bits of each word are flags saying which of its registers the call
clears after pushing them (arguments that are dead temporaries, from the
lifetimes the code generator computes).  Operand words are not opcodes:
everything that walks the code skips them by position, and the verifier
checks that no jump lands on one.  A return is `tailcall r0 (results)`.

A trailing vararg or call result keeps the one-by-one pushes (`push r, ...r`),
as do calls of more than 32 arguments, since listed arguments hold their
registers until the call.  The etc bit on the count described under Etc
values is part of step 3.

Measured (7 November, one binary with the old pushes behind a switch, 6
rounds): executed opcodes −13.8% geomean (md5 −26%, towers −24%, queens
−21%), time −2.0% (towers −7.0%, md5 −5.6%, queens −5.5%, spectral-norm
−4.0%).  A push was a cheap opcode and the work it did still happens inside
the call; what was saved is the dispatch.

## Pushing values

Since step 1, every value a call lists is pushed by one loop inside the call
opcode, so the receiver's type can be checked once per call rather than once
per value.  Measured variants (call microbenchmarks, best of 4 interleaved
runs, ns):

| | before | direct `Push` | store written out | one helper call |
|---|---|---|---|---|
| `f(i)`, 1 result | 60.5 | 61.4 | 58.5 | 63.2 |
| `f(i, i, i)`, 1 result | 70.4 | 71.7 | 65.2 | 69.7 |
| 3 results | 77.5 | 78.0 | 72.0 | 74.9 |

- Calling `(*LuaCont).Push` directly instead of through `Cont` gains
  nothing: a predictable interface call costs no more than a direct one.
- The compiler can't inline `Push`, as each call it can't inline costs 57 of
  its budget of 80 and the varargs case has to call out.  Writing the common
  case out in the call opcode (the receive opcode puts the value in a
  register: store it there) saves about 1.4 ns a value.
- Pushing all the values in one call to a helper loses for one or two
  values, which most calls have: the helper takes 13 words of arguments and
  tests the receiver's type, which costs more than the batching saves.

The store written out is the change taken (#229): suite geomean −1.77%,
md5 −5.4%, queens −4.0%, permute −3.8%, list −3.7%, json −2.4%, nothing
slower.

**Parameters skipping the receive opcodes: tried, not worth it.**  Of the
474 functions in the benchmark programs, 414 (87%) begin `recv r1, recv r2, ... recv rn`: plain
registers, in order; 34 have no parameters, 24 receive a captured parameter
into a cell, 2 end with `recv ...r`.  The length of that leading block is
worked out at load (`Code.paramRegs`, no format change), and an argument
arriving at a pc below it goes to the register after the pc, without reading
the opcode.  The test is exact: the pc is below `paramRegs` only while the
frame takes its arguments, as every other position, including after any
call, lies beyond the block.  Results going back into a caller keep the
general path; they go to the one `recv` after the call, anywhere in the
frame.

Measured on top of the store written out, it was slightly slower
(`f(i, i, i)` 65.6 against 65.0 ns, 3 results 72.8 against 71.4, suite
−1.23% against −1.77%): the receive opcode is next to the frame and cheap to
read, and the extra test is paid by every value, results included, which
never take the shortcut.  Kept here so that it is not tried again for the
same reason; it would only pay if the test were free, for instance if the
call knew at its site that its callee takes its arguments this way.

## Receiving: a dense `recv` (step 2, reconsidered)

The plan was to let one `recv` word name up to three registers (`0000 F a b
c | A | B | C`), F meaning that the last of them is an etc that collects the
rest, so a function starts with `recv r1, r2, r3`, `local x, y = f()` is
`call rF` then `recv r5, r6`, and `function(a, ...)` begins `recv r1, ...r2`.
`Push` would find the k-th destination at the continuation's pc, register k
of the current word, stepping to the next word every three values.

It was expected to save code words and no time, as `recv` never executes for
a value that arrived, and the attempt above confirms that reading a receive
opcode is cheap.  It would also make the store written out in the call
opcode find a register within a word, which needs a counter on the
continuation.  So it comes last, if at all.

Captured parameters are received into their cells as today.

## Step 3: the callee in the call

`call rF, n` is a Type 5 opcode (`OpCall`) with F meaning tail call, its
16-bit field holding `n`, the number of arguments; the top bit of the low byte
says that the *last* listed register is an etc whose values are all pushed.

**rF holds the callee, not a continuation.**  The compiler makes a
continuation in exactly one place and always calls it at once, so the call
creates it: if `rF` holds a continuation (the case of `r0`, a return) it is
used as is; otherwise `rF` is a callable value (a closure, a Go function, or a
value with `__call`) and a continuation for it is made.  One type test per
call.  The callee's continuation never lives in the caller's frame, which
also removes the `clearReg` of the continuation register on the way out.

That type test is already where it would go: since #225 the `cont` opcode
makes the frame with a type switch on the callee, a direct `NewLuaCont` or
`NewGoCont`, written out in the run loop since the compiler can't inline
`Continue` (each call it can't inline costs 57 of its budget of 80, and
`Continue` makes three).  Step 3 moves that switch into the call and drops
the `cont` dispatch, the register write and its clear.

**And the switch to the callee.**  Once the call makes the callee's frame,
it could also run it: for a Lua function, and with no debug hooks, the call
would swap the run loop's locals to the callee's frame and go on, instead of
returning it to the trampoline; a return into a Lua caller likewise.  That
takes steps 3 to 5 of What a call costs now out of every Lua-to-Lua call and
return, twice per call.  The model is unchanged: frames are still
continuations, the Go stack does not grow, and Go functions, hooks and
errors still go through the trampoline.  What it needs:

- The trampoline attributes an error to the continuation it called; with
  frames switching inside `RunInThread` it must read `t.currentCont`, which
  the switch sets.
- The pending-finalizer check moves into the switch too (one load).
- The frame left is marked not running and the one entered running.
- `RunInThread` is sensitive to layout (edits to it have moved untouched
  benchmarks by 1 to 3%), so this is measured on its own, against the
  benchmarks that make few calls as canaries.

This is a second way for control to pass between frames, which is its
cost.  It is worth a prototype to measure the gain before it is designed in
detail.

## Etc values

Sending: in `f(a, ...)` or `f(a, g())` the last listed argument register
holds an etc (the frame's vararg register, or a register that collected a
call's results) and the etc bit of `n` is set.  The call pushes the fixed
arguments, then expands the etc and pushes each of its values, as `pushetc`
does today.  Lua expands only the last argument, so one bit is enough.

Receiving: a `recv` with F set collects into its register every value beyond
the fixed ones, as today.  So `print(f())` would be

    call rF, 0                    -- f's results...
    recv ...rT                    -- ...all into rT
    call rPrint, 1|etc (rT)       -- push them all

## What stays in the instruction set

`OpCont`, `OpTailCont`, `OpEtcId` (`pushetc`) and `OpCC` remain; the compiler
stops emitting the first three in step 3.  First-class continuations, if
they come, go through `OpCC` (the current continuation as a value), which
this design does not touch.

## Compare-and-jump (step 4)

`jumpif` is the most executed opcode after the call protocol and table
access (11.2% of the census above on average, 20 to 27% in list, sieve and
mandelbrot), most preceded by a comparison or a `not`.  A fused
compare-and-jump needs two registers, an operator and an offset.  Type 4a
(`0101 F a b 1 | A | B | Z`) has the two registers and an 8-bit Z with many
unused values for the operator (and F for its negation); the 16-bit offset
goes in one operand word after it, costing no dispatch.  So compare-and-jump
does not need a free type prefix, which is why `recv` can stay.  It needs
its own note.

## Order of work

Each step keeps the conformance suite and both test builds green and is
measured on its own (opcode counts, then the interleaved A/B).

1. **Arguments on the call**: done, #210.
2. **Pushing values**: the store written out (#229).
3. **Callee in the call**: `rF` is the callee; the etc bit on `n` replaces
   `pushetc`; the compiler stops emitting `cont` and `tailcont`.  Then, on
   its own, the switch to the callee without the trampoline, prototyped
   first.
4. **Compare-and-jump** on Type 4a with an operand word for the offset.
5. **Dense `recv`**, if code size comes to matter.

## Details to settle

- `DebugInfo` subtracts one from the pc of a non-running frame to find the
  current line; after a call that is the call's last operand word, which
  carries the call's line, so this works, but it is worth a test once the
  call makes the callee's frame.
- Resolving `__call` inside the call opcode can raise an error; the frame's
  pc must be set before, as for other errors in the run loop, so that the
  message names the right line.
- The debug hooks for calls and tail calls receive the callee's continuation,
  which the call will create; the order (create, push, hook, transfer) must
  match today's.
- `GoCont` keeps its own `Push` (it collects arguments in a slice, and its
  `Push` already inlines); the call opcode's written-out store is for
  `LuaCont` only.
- Binary chunks compiled before step 3 contain `cont` opcodes, which remain
  in the instruction set.  No signature bump is needed.
