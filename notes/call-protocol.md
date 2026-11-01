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

### A continuation's receive spec

Every continuation has a *receive spec*: the list of registers the next pushed
values go to, with an optional etc register for the rest.  It is written as a
call word (see below) with `n = 0` arguments and `m` result registers listed in
operand words:

- For a fresh frame, the spec is **opcode 0 of the function**, a call word that
  is never executed as a call: the frame starts just after it.  The
  disassembler shows it as `recv (r1, r2, ...r3)`.  This replaces the `recv`
  opcodes with one word (plus operand words) and needs no new header field, so
  the marshalled format of a `Code` is unchanged.
- For a frame waiting on a call, the spec is **the call word itself**, whose
  result list says where the results go.  This replaces the `recv` opcodes
  that followed a call.

A `LuaCont` therefore remembers one program counter, the position of the call
word that is its current spec (0 at creation, the call it made otherwise), and
how many values it has received.  `Push` computes the k-th destination from the
operand words, which is cheaper than reading and advancing past a `recv`.  When
the frame resumes, `RunInThread` sets the registers that received nothing to
nil, and the etc register to the collected rest, once, before running; this is
the same code for entering a function and for returning into one.

A captured parameter lives in a cell.  The spec lists value registers only, so
the compiler emits `uK <- rJ` after the entry word for each captured
parameter.  Across the benchmarks 5% of parameters are captured.

### The call word

`call rF, n, m` is a Type 5 opcode (`OpCall`) with F meaning tail call.  Its
16-bit field holds `n`, the number of arguments, and `m`, the number of
results, one byte each.  The top bit of each byte says that the *last* listed
register of that list is an etc: its values are all pushed (arguments), or it
collects the rest (results).  So the compiler can list up to 127 of each; it
lists at most 32 arguments and pushes one by one beyond that, because listed
arguments hold their registers until the call.

The operand words follow, `ceil((n+m)/3)` of them, listing the argument
registers then the result registers, three per word in the A, B, C positions
with their kind bits.  The remaining bits of each word are flags saying which
of its registers the call clears after pushing them (arguments that are dead
temporaries), as in #176.  Operand words are not opcodes: everything that
walks the code skips them by position, and the verifier checks that no jump
lands on one.

**rF holds the callee, not a continuation.**  The compiler makes a
continuation in exactly one place and always calls it at once, so the call
creates it: if `rF` holds a continuation (the case of `r0`, a return) it is
used as is; otherwise `rF` is a callable value (a closure, a Go function, or a
value with `__call`, resolved as `cont` resolves it today) and a continuation
for it is made.  One type test per call.  The callee's continuation never
lives in the caller's frame, which also removes the `clearReg` of the
continuation register on the way out.

A return is `tailcall r0, n, 0`.  Entering a function has no opcode.

### What stays in the instruction set

`OpCont`, `OpTailCont` and `OpCC` remain; the compiler stops emitting the first
two.  First-class continuations, if they come, go through `OpCC` (the current
continuation as a value), which this design does not touch.  `Type 0` is no
longer emitted and becomes free, see Payoff.

## Payoff

Executed opcodes, from the counts at 17a888f: pushes are 13 to 23% of the
object-oriented benchmarks (#176 takes most of them), `cont`/`tailcont` 5 to
8% more, `recv` 0% (it never ran).  Code size drops by the `recv` words.

The strategic payoff is the freed type prefix.  `jumpif` is 11 to 23% of the
executed opcodes in every benchmark and most are preceded by a comparison or a
`not`; a fused compare-and-jump needs two registers and an offset, which no
existing type can hold.  Type 0 can.

## Order of work

Each step keeps the conformance suite and both test builds green and is
measured on its own (opcode counts, then the interleaved A/B).

1. **Arguments on the call** (#176): `call rCont, n` with operand words,
   clear flags, scanners skip by position.
2. **Receive specs.** The call word gains `m` and the result list; functions
   start with an entry word; the compiler stops emitting `recv`; `Push` and
   the nil fill at resume use the spec.  Expected: no time change, fewer code
   words, simpler `Push`.  The verifier checks result registers and the entry
   word; the disassembler prints the entry word as `recv (...)`.
3. **Callee in the call.**  `rF` is the callee; the etc flag on the argument
   list replaces `pushetc`; the compiler stops emitting `cont` and `tailcont`.
   Expected: 2 to 3% on the object-oriented benchmarks.
4. **Compare-and-jump** on the Type 0 prefix.  Needs its own note once 2 and 3
   are in.

## Details to settle while doing step 2

- The frame's pc at creation is `1 + ceil(m/3)`; `DebugInfo` subtracts one
  from the pc of a non-running frame to find the current line, which must
  skip operand words too.
- The nil fill at resume needs the count of received values; `Push` keeps it
  on the `LuaCont`.  The etc register gets `ArrayValue(acc)` at resume, as
  `recvetc` does today when it executes.
- `GoCont` keeps its own `Push` (it collects arguments in a slice); the
  symmetry of the two `Push` implementations is worth keeping readable.
- Calls from Go into Lua (`Call`, coroutine resume) push into a fresh
  `LuaCont` and work from its entry word unchanged.
- Binary chunks compiled before step 2 contain `recv` opcodes that step 4 will
  reinterpret.  Bump the binary chunk signature at step 2 so that old chunks
  are refused by `load` rather than misread.
- The entry word for a function with no parameters is a single word with
  `m = 0`; every function has one, so the verifier's rule is uniform.
