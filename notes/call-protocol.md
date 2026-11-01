# The call protocol: design note

Status: design, 1 November 2026.  Step 1 is PR #176; the rest is not written.

## Where this comes from

Counting the opcodes the benchmarks execute (see [ir-passes.md](ir-passes.md))
put the call protocol at 30 to 45% of all opcodes in the object-oriented
programs.  A call `o:m(a, b)` compiled to

    r4 <- r2[K ("m")]
    r4 <- cont(r4)          -- make the callee's continuation
    push r4, r2             -- self
    push r4, r5
    push r4, r6
    call r4

and the callee began with `recv r1; recv r2; recv r3`.  Eight opcodes of
protocol for one call, and the measurements say that on these programs an
opcode costs about the same whatever it does: the trip through the run loop
is the price.  The protocol can be two words.

## The model

golua is a continuation machine: there is no stack, a frame is a
continuation, and a return is a call of the caller's continuation, which
every frame holds in `r0`.  Values move between frames by being *pushed* into
a continuation; the receiving frame says where they land.  Today it says so
with `recv` opcodes at the point where it resumes: at pc 0 for its parameters,
after a `call` for the call's results.  A push into a `LuaCont` reads the
`recv` at the continuation's pc, writes the value there and advances the pc;
when the frame runs, any `recv` still ahead of the pc sets its register to nil.
That is why `recv` never shows in the dynamic counts: it only executes for
values that were not supplied.

The redesign keeps the model and changes only how the two sides are written
down.

### Receiving: a dense `recv`

A continuation's *receive spec*, the registers the next pushed values go to,
stays where it is today: `recv` opcodes at the point where the frame resumes,
at pc 0 for its parameters and after a `call` for the call's results.  What
changes is density.  Type 0 has three register slots (`0000 F a b c | A | B |
C`), so one `recv` word names up to three registers, with F meaning that the
last of them is an etc that collects the rest; a function or a call needing
more chains a second word.  The disassembler shows `recv r1, r2, ...r3`.

`Push` keeps its mechanism: it finds the k-th destination at the
continuation's pc, now register k of the current word, stepping to the next
word every three values.  `recv` words beyond the values received execute
when the frame resumes and set their registers to nil (or the etc to an empty
array), as the leftover `recv`s do today, so `recv` still never executes for
a value that arrived.  A function starts with `recv r1, r2, r3`; a return is
`tailcall r0, n`; `local x, y = f()` is `call rF, n` then `recv r5, r6`.

Captured parameters are received into their cells as today.  An earlier draft
of this note replaced `recv` with an entry word at opcode 0 and result lists
on calls, to free the Type 0 prefix for compare-and-jump; operand words give
compare-and-jump a home without that (see Payoff), so the simpler design wins.

### The call word

`call rF, n` is a Type 5 opcode (`OpCall`) with F meaning tail call.  Its
16-bit field holds `n`, the number of arguments; the top bit of the low byte
says that the *last* listed register is an etc whose values are all pushed.
The compiler lists at most 32 arguments and pushes one by one beyond that,
because listed arguments hold their registers until the call.

The operand words follow, `ceil(n/3)` of them, listing the argument registers
three per word in the A, B, C positions with their kind bits.  The remaining
bits of each word are flags saying which of its registers the call clears
after pushing them (arguments that are dead temporaries), as in #176.  Operand
words are not opcodes: everything that walks the code skips them by position,
and the verifier checks that no jump lands on one.

**rF holds the callee, not a continuation.**  The compiler makes a
continuation in exactly one place and always calls it at once, so the call
creates it: if `rF` holds a continuation (the case of `r0`, a return) it is
used as is; otherwise `rF` is a callable value (a closure, a Go function, or a
value with `__call`, resolved as `cont` resolves it today) and a continuation
for it is made.  One type test per call.  The callee's continuation never
lives in the caller's frame, which also removes the `clearReg` of the
continuation register on the way out.

A return is `tailcall r0, n`.

### Etc values

Sending: in `f(a, ...)` or `f(a, g())` the last listed argument register
holds an etc (the frame's vararg register, or a register that collected a
call's results) and the etc bit of `n` is set.  The call pushes the fixed
arguments, then expands the etc and pushes each of its values, as `pushetc`
does today.  Lua expands only the last argument, so one bit is enough.

Receiving: a `recv` word with F set collects into its last register every
value beyond the fixed ones, as `recvetc` does today.  So `print(f())` is

    call rF, 0                    -- f's results...
    recv ...rT                    -- ...all into rT
    call rPrint, 1|etc (rT)       -- push them all

and `function(a, ...)` begins with `recv r1, ...r2`.

### What stays in the instruction set

`OpCont`, `OpTailCont`, `OpEtcId` (`pushetc`) and `OpCC` remain; the compiler
stops emitting the first three.  First-class continuations, if they come, go through `OpCC` (the current
continuation as a value), which this design does not touch.  `Type 0` is no
longer emitted and becomes free, see Payoff.

## Payoff

Executed opcodes, from the counts at 17a888f: pushes are 13 to 23% of the
object-oriented benchmarks (#176 takes most of them), `cont`/`tailcont` 5 to
8% more, `recv` 0% (it never ran, so the dense form saves code words, not
time).

The next counted item is `jumpif`, 11 to 23% of the executed opcodes in every
benchmark, most preceded by a comparison or a `not`.  A fused compare-and-jump
needs two registers, an operator and an offset.  Type 4a (`0101 F a b 1 | A |
B | Z`) has the two registers and an 8-bit Z with many unused values for the
operator (and F for its negation); the 16-bit offset goes in one operand word
after it, costing no dispatch.  So compare-and-jump does not need a free type
prefix, which is why `recv` can stay.

## Order of work

Each step keeps the conformance suite and both test builds green and is
measured on its own (opcode counts, then the interleaved A/B).

1. **Arguments on the call** (#176): `call rCont, n` with operand words,
   clear flags, scanners skip by position.
2. **Dense `recv`**: up to three registers a word, F for the etc; `Push`
   indexes within a word; `ircomp` packs the `Receive` and `ReceiveEtc`
   instructions; verifier and disassembler follow.  Expected: no time change,
   fewer code words.
3. **Callee in the call.**  `rF` is the callee; the etc bit on `n` replaces
   `pushetc`; the compiler stops emitting `cont` and `tailcont`.  Expected:
   2 to 3% on the object-oriented benchmarks.
4. **Compare-and-jump** on Type 4a with an operand word for the offset.  Needs
   its own note once 2 and 3 are in.

## Details to settle while doing steps 2 and 3

- `Push` into a `LuaCont` must know how many values the current `recv` word
  has taken so far; a small counter on the continuation, reset when the pc
  moves to the next word.
- `DebugInfo` subtracts one from the pc of a non-running frame to find the
  current line; after a call that is now the call's last operand word, which
  carries the call's line, so this still works, but it is worth a test.
- `GoCont` keeps its own `Push` (it collects arguments in a slice); the
  symmetry of the two `Push` implementations is worth keeping readable.
- Resolving `__call` inside the call opcode can raise an error; the frame's
  pc must be set before, as for other errors in the run loop, so that the
  message names the right line.
- The debug hooks for calls and tail calls receive the callee's continuation,
  which the call now creates; the order (create, push, hook, transfer) must
  match today's.
- Binary chunks compiled before step 2 contain single-register `recv`
  opcodes, which remain valid (a word with one register); chunks compiled
  before step 3 contain `cont` opcodes, which remain in the instruction set.
  No signature bump is needed for these steps.
