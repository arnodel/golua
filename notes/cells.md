# Cells: making them explicit

Status: design, 9 November 2026, revised 10 November; built, 11 November,
as a stack of PRs (#233, #234, #235), each step measured (see the end).  It
follows #230 (fixed registers) and #231 (the call makes the callee's
frame).

A local that a closure captures lives in a *cell*, so that the function and
the closure share it.  Today a cell is a kind of register, at every level:
the IR, the opcodes, the frames.  This note proposes that a register only
ever holds a value, and that a cell be a value of its own, a *box*, which
registers hold and which explicit instructions read and write.  The point
is that each concept then means one thing; the speed it may bring is
secondary, and is to be measured.

This is a known technique, *assignment conversion*: compilers for Scheme
box the captured variables that are assigned, and capture the others by
value.  Here the analysis that decides which is which already exists
(`ir/fixed.go`).

It splits into two halves that can be done and judged separately: the IR
half, which changes no opcode, and the run-time half, which changes the
opcodes and the frames.

## Where we are

**In the IR**, a register has `IsCell` set when a closure captures it.  The
AST compiler decides that after the fact: it emits instructions for a local
as for any other, and when a nested function refers to it, `getRegister`
marks the register a cell, which turns the instructions already emitted
into cell operations.  A cell register stands for two things:

- *the box*: `clr u0` (`ClearReg`) puts a new one in the register, which a
  local declared in a loop needs each time round;
- *its contents*: `u0 <- 0` and `u0 <- r7` are stores into the box, and a
  read of `u0` reads what the box holds.

So "how many times is this register written" has no clear meaning for a
cell: a store into the box and a new box are both writes.  `ir/fixed.go`
has to say that a clear "does not count, it starts a new cell", and the
pass invariants (`notes/ir-passes.md`, rule 3) treat cells as a special
case throughout.

**In the opcodes**, every register operand carries a cell bit (bits 26, 25
and 24 for A, B and C).  So the meaning of an operand depends on a bit
that most operands leave at 0.

**In the run loop**, reading an operand tests that bit:

```go
func getReg(regs []Value, cells []Cell, reg code.Reg) Value {
	if reg.IsCell() {
		return *cells[reg.Idx()].ref
	}
	return regs[reg.Idx()]
}
```

and `setReg` does the same for a write.  It is the commonest operation of
the run loop.  Both are inlined at every operand of every opcode, so every
case of the run loop carries its own copy of the cell path.  On the build
of 10 November 2026, `RunInThread` is 4093 instructions.  About 170 of
them are cell tests (some 60 sites) and about 450 the cell paths: together
about 600, a seventh of the loop, as many as the plain register paths.

**In the frames**, a `LuaCont` has `cells []Cell` beside its registers: the
closure's upvalues first, then a new cell for each of its own captured
locals, made at every call (`newCell(NilValue)` allocates the value it
points to), and again at every `clr`.  A `Closure` holds `Upvalues []Cell`,
which the `upval` opcode fills after `clos` makes it.

## The model

Three kinds of thing, each meaning one thing:

- **a register** holds a value, and only its function reads or writes it;
- **a box** is a value (a reference to a value) that a register can hold,
  made by a `newbox` instruction;
- **the contents of a box** are read and written by `load rV, [rB]` and
  `store [rB], rV`, as a table field is by `r[k]`.

A captured local is a register that holds a box, made where the local
comes into existence (today's `clr` is that `newbox`).  Its uses are loads
and stores.  The register holding the box is written once per box: a plain
register like any other, and in a loop a changing one, since each time
round makes a new box, which is as it should be.  A closure captures the
box, which becomes its upvalue.

The one rule the passes need about boxes is the rule they already have
about tables: a store into any box, or a call, may change the contents of
any box, so a value loaded from a box is only known until the next one.

What this makes clear:

- every register is a value register, and "fixed" (`ir/fixed.go`) for a
  register is about its writes only;
- "fixed" for a captured local becomes a property of a box, "its contents
  are stored before anyone can load them, and never after": the same
  dataflow, on loads and stores, where a `newbox` starts a new object.
  That is today's "a clear starts a new cell", said in terms of what it
  means;
- named locals that are not captured are plain registers that could be
  renamed at each assignment (static single assignment), after which a
  register written once is the common case rather than one to prove.
  Rule 5 of `notes/ir-passes.md` (named locals stay observable) would
  then need a mapping from names to registers for the debug library;
- the passes need no rule about cells beyond the memory rule above.

## When the decision is made

The AST compiler cannot know, while it compiles a function, which of its
locals will be captured: a nested function later in the body may capture
one.  So the front end goes on emitting locals as registers, as now, and
once the chunk is compiled a lowering step turns each captured local into
a box register with loads and stores.  Its input is today's IR with
`IsCell`; its output is IR without it.

The lowering needs only `IsCell`.  Fixedness is then worked out on the
lowered IR, over loads and stores, by the dataflow of `ir/fixed.go` with
`newbox` in the place of the clear.  This order matters: done the other
way round, with the lowering after `markFixed`, the analysis would still
run on cell registers with the clear rule, and the model would gain
nothing there.

## The IR half

The lowering emits `newbox`, `load` and `store` in the IR, and the passes
run on the lowered IR.  The code generator (`ircomp`) maps each box
register to a cell slot of today's frames, turns `newbox` into `clr`, a
load into cell operands at its uses (`load r, [b]; r <- r + r6` becomes
`r <- u0 + r6`), and a store into a cell destination.  So the opcodes stay
those of today: no format change, no change to the run loop or the Go API.

The condition for folding a load is not how many uses it has but what
lies between it and them: nothing that can write the box, which by the
memory rule means no store into any box and no call.  `load r, [b]; r <-
r + r` folds at both uses; `load r, [b]; call f; r <- r + 1` does not, and
keeps its load as a cell-to-register move, which is what the compiler
emits today for a cell read across a call.  That is a small alias
analysis, and it lives in `ircomp`.

What it gives:

- the IR model above, and passes and `ir/fixed.go` with the memory rule in
  place of the cell rules;
- writing straight into a cell (`r7 <- u0 + r6; u0 <- r7` becoming
  `u0 <- u0 + r6`) falls out of the folding;
- a check: on the benchmarks, the opcodes executed should be the same as
  before, apart from the folding of stores.

What it does not give: a run-time gain.  It moves the knowledge of cells
from the passes and the analysis into one place, the folding, rather than
removing it.  Its worth is the model, and that it makes the run-time half
below measurable.

## The run-time half

- The operands lose their cell bit: `getReg` and `setReg` become `regs[i]`.
  The three bits are free for other uses (a constant-operand flag, more
  opcodes, flags like the argument-clear bits of calls).
- New opcodes: `newbox rB`, `load rV, rB`, `store rB, rV`.  A frame no
  longer needs `cells []Cell` for its own locals: a box lives in a
  register.
- A closure's upvalues are boxes, as its cells are now.  When `NewLuaCont`
  builds a frame, it puts each upvalue into a register.  For a changing
  upvalue that is the box, read by `load` opcodes.  For a fixed upvalue
  there are two forms, to measure against each other:
  - *eager*: the frame puts the box's *contents* in the register, a load
    at the function's entry done in Go without a dispatch, valid because
    the contents cannot change during the call.  It costs a pointer chase
    per upvalue per call, whether or not the call uses it;
  - *lazy*: the box, as for a changing upvalue, with `load` opcodes where
    it is used, folded away wherever the IR half's condition allows.  It
    costs a dispatch per use that cannot fold.
- In the function that owns a fixed box, the value it stores is still in
  a register, so its own reads use that register, as copy propagation does
  for fixed cells today.
- Boxes and upvalues share the function's registers with its locals and
  temporaries.  Today cells have 256 slots of their own; a function near
  the limit of 256 registers that compiles today might not.  The frame's
  register pool class grows by the upvalues the function uses.
- Binary chunks change format; the verifier checks the new opcodes.
- The Go API changes where it speaks of cells: `Closure.AddUpvalue(cell
  Cell)`, `GetUpvalue`, `SetUpvalue` (`runtime/closure.go`).

## What the run-time half trades

**What it removes.**  With no cell bit, `getReg` is one indexed load and
`setReg` one indexed store: every operand of every opcode becomes
straight-line code, with no branch before the value is read, so the
processor can start the reads earlier.  The run loop loses its 60 or so
cell tests, taken several times per opcode, and the seventh of its code
that the cell paths are.  The run loop's speed is sensitive to its layout
(untouched benchmarks move 1–3% when it changes), so its size counts on
its own.

**What it adds.**  A captured local used as an operand today costs
nothing extra: `u0 + r6` reads the cell in the add.  With boxes it costs a
load first.  `y = y + 1`
on a captured `y` is two opcodes today (`r7 <- u0 + r6; u0 <- r7`, one if
copy propagation wrote straight into cells) and three with boxes (load, add,
store).

Only cells that change pay this.  A fixed upvalue is in a register from
the start of the call (eager) or loaded once and folded (lazy), and a
fixed local of the function's own is in the register it was stored from,
so their reads are plain register reads and need no `load`.  Captured variables are mostly functions and modules,
which are fixed: the census of #230 found every call through a cell in the
benchmarks to be through a fixed one.

The commonest case is `_ENV`.  The program never creates it: `load` makes
its box and gives it to the main chunk as upvalue 0, and every function
that uses a global captures it from there.  In almost every chunk nothing
assigns `_ENV`, so it is fixed in all of them, and each function that uses
globals starts with `_ENV` in a register: `u0[K]` becomes `r[K]`, with no
`load`.  The box stays, because the debug library names it: changing a
chunk's `_ENV` with `debug.setupvalue` or `debug.upvaluejoin` (the
replacement for `setfenv`) still reaches every function the chunk made.

Each added opcode goes through the run loop's dispatch, the one branch the
processor predicts badly.  So the trade is many well-predicted branches on
every opcode against more dispatches where a changing cell is used, and
only a prototype can say which wins.

Putting upvalues into registers has a cost of its own: today a function
with no captured locals of its own borrows its closure's upvalues as its
cells at no cost (`borrowCells` in `NewLuaCont`).  The copy is one box
per upvalue, or one pointer chase for an eager fixed upvalue, in Go, far
less than a dispatch; the prototype says whether eager or lazy wins.

**The deviation.**  "Fixed" is a property of the program as written, and
`debug.setupvalue` can break it while a function runs.  With fixed
upvalues read at the start of the call, a call that is running when its
upvalue is changed that way goes on seeing the old value until it returns;
calls after it see the new one.  So a chunk whose `_ENV` is changed while
it runs keeps the old environment until it returns.  The usual idiom sets
the environment of a freshly loaded chunk before running it, which behaves
as in Lua.  This needs a line in `CONFORMANCE.md`.  Only the eager form
has it; the lazy form reads the box at each use, as today.  (#230 is
already exposed, barely, and in the other direction: it reads a fixed
cell at the call rather than before the arguments, so an argument
expression that changed the callee's upvalue through the debug library
would be seen sooner than in Lua, not later.)

## Capturing fixed values by value

The step further: since a fixed box's contents are set before any closure
can see them, a closure could capture the *value* instead of the box.  No
box would be made for a fixed local, and an upvalue read would be a read of
a value in the closure.  A local function that calls itself captures its own
box before the box holds it; the closure would capture itself instead,
which works because `upval` runs after `clos`.

But the debug library sees upvalues as shared variables, and the
conformance suite tests it:

- `debug.upvalueid` must return the same identity for two closures that
  capture the same local, and different ones otherwise (`closure.lua`
  256–263, `goto.lua` 203–231).  In `closure.lua`, `a` and `b` are fixed,
  and `foo1` and `foo2` must share them.
- `debug.upvaluejoin` makes one closure's upvalue refer to another's
  (`closure.lua` 269–274), and `debug.setupvalue` changes the variable for
  every closure that shares it.

With values copied into each closure, neither holds.  The identity would
need an object made at capture time, which is the box again.  So capturing
by value is a separate decision with a semantic cost: a deviation to
document in `CONFORMANCE.md` and adapt tests for, or not done.  The box
model above stands without it.

It would also gain less than it seems: with fixed upvalues in registers
from the start of each call, or loaded once and folded, reads already
cost nothing.  What it would
still save is the box allocated for each fixed local that is captured,
which the allocation count below measures.

## To measure

1. **Loads the run-time half would add**: executed opcodes with an operand
   that is a changing cell, with the census tooling of #230.  Operands that
   are fixed cells, `_ENV` among them, add none.
2. **Allocations**: cells made per benchmark run, at frame creation and at
   `clr`.
3. **The cell test**, roughly: a second, always-false test in `getReg` and
   `setReg` gives the order of magnitude of one branch per operand.  Layout
   effects make it rough.
4. **The trade itself**, with a prototype built on the IR half: the code
   generator does not fold loads, so cells are touched only by `load` and
   `store` opcodes, and every other `getReg` and `setReg` loses its
   branch.  That build runs real programs, so the suite measures the extra
   loads against the branches and code removed.  A second build with the
   eager form for fixed upvalues measures eager against lazy.

## Order of work

1. The counts (1 and 2 above).
2. The IR half.  Worth doing for the model alone, and it makes the
   run-time half measurable.
3. The prototype of the run-time half (4 above), lazy then eager.  If it
   wins, it becomes the real format change: new opcodes, operands without
   the cell bit, frames without own cells, the verifier and the Go API.
   If it loses, the IR half stands on its own.
4. Capturing by value, only if the debug-library deviation is accepted.

## Measured: the prototype, 10 November 2026

`ir.LoadStoreCells` (commit 8182359 on this branch) is measurement 4 in its
lazy form with nothing folded: every instruction with a cell operand gets a
plain temporary and a move from or into the cell, and the run loop indexes
registers directly everywhere but in the move and `upval` opcodes.  Against
`lua5.5` (bf58b12), `-tags noquotas`, M1 Max, six rounds gated on a quiet
machine:

- **Opcodes executed: +2.92% geomean** (binary-trees +8.2, richards +8.1,
  havlak +7.7, cd +7.4, md5 +7.3; nine benchmarks unchanged).  The moves
  the folding and the eager form would remove.
- **Time: −3.82% geomean**, nothing significantly slower.  fannkuch −12.1,
  sieve −8.7, spectral-norm −6.5, life −6.0, queens −5.9, md5 −5.5,
  permute −5.2, mandelbrot −4.7, fasta −4.4, nbody −4.1, list −3.9,
  bounce −3.7, towers −3.5; cd, deltablue, havlak, richards, binary-trees
  and regex-dna within noise.
- **Microbenchmarks** (best of four rounds): loops over plain registers
  −12%; the cell shapes pay for their unfolded moves: `_ENV[K]` +4%,
  fixed upvalue read +11%, counter in a closure +6%, calling a closure that
  increments its upvalue +8.5%.  Calls through a local or a method −4 to
  −5%, through an upvalue +1 to +3%.

So the branch on every operand and the seventh of the loop cost more than
3% more dispatches, by a clear margin, and the trade is settled before any
folding.  What is left of the added opcodes is what the IR half's folding
and the eager form of fixed upvalues take back.

## Measured: the fold and the entry loads, 10 November 2026

Two more steps on the same branch, each against `lua5.5` in the same gated
run as a rerun of the prototype:

1. **The fold** (`ir: fold the loads and stores of cells`): a fixed upvalue
   loaded once at entry, and within a stretch a register known to hold a
   cell's contents standing for the cell, until it is written or, for a
   cell that is not fixed, until an instruction that may run code.  The
   code generator drops a move between two registers given the same slot.
   Opcodes +2.18% (from +2.92%); coroutine-ring, spectral-norm and life to
   0, md5 +7.3 → +3.4, richards +8.1 → +5.4.  But the entry load is an
   opcode per call whether the call uses the upvalue or not, so bounce and
   towers went up (+1.1 → +2.9, +2.1 → +4.2), and the calls through fixed
   upvalues stayed (binary-trees +8.2, cd +7.9, havlak +6.8).  Time −3.46%.
2. **The entry loads** (`a fixed upvalue is loaded when the frame is
   made`): the load of a fixed upvalue at entry becomes `code.EntryLoad`,
   a copy `NewLuaCont` does, with no opcode.  The pass puts those loads
   before the parameters' `recv`, so their registers are not ones the
   `recv` writes.  Opcodes **+0.06%** (richards −2.0, cd −0.9, fasta +4.1
   for its changing upvalues `Last` and `nout`, everything else 0).  Time
   **−4.24%**: fannkuch −12.2, spectral-norm −9.0, sieve −8.7, md5 −7.4,
   life −6.7, queens −5.5, permute −5.4, mandelbrot −5.0, nbody −4.7, list
   −3.8, towers −3.7, bounce −3.4, fasta −3.4, richards −2.6, deltablue
   −2.3, storage −2.3, havlak −2.0, k-nucleotide −1.9, json −1.8,
   binary-trees −1.8; cd and coroutine-ring within noise; regex-dna +0.4%.

Microbenchmarks with the entry loads: loops over plain registers −16%, a
fixed upvalue read in a loop −16% (from +10% in the prototype), `_ENV[K]`
−10%, calls −4 to −12%, a counter in a closure −1%, calling a closure that
increments its upvalue 0%.  So a fixed upvalue is now exactly a plain
register, and the one shape the model taxes, a changing cell, comes out
even.

## Measured: the format change, 11 November 2026

The real format change (#235): `code.Reg` is an 8-bit index and nothing
else; `load`, `store` and `newbox` opcodes, `upval` taking a box register;
a box made where its local is declared; a frame given the boxes it uses
(or the contents of a fixed upvalue it only reads) when it is made; no
`CellCount`, no cells in frames, no cell pool; the kind bits reserved and
zero.  The IR gets `Load`, `Store` and `NewBox` first, in a commit that
changes no output.

Against the stack top (#234), opcodes **+0.09%** (havlak +1.5, deltablue
+0.6: `newbox`es that were allocations when the frame was made), time
**−2.76%**: md5 −5.3, permute −5.2, towers −5.0, spectral-norm −5.0,
queens −4.2, bounce −3.5, list −3.4, deltablue −3.2, storage −3.0,
fannkuch −2.8, json −2.7, havlak −2.5, richards −2.4; k-nucleotide,
regex-dna and life within noise; nothing slower.  That is the price of
the kind bit, which every register access tested.  Against #231 the whole
cells change is opcodes +0.30%, time **−6.91%**.

Microbenchmarks against the top: calls −3.5 to −5.9%, loops and fixed
upvalue reads unchanged.  One shape pays: writing a changing upvalue in a
loop, +8.7% (21.8 → 23.7 ns per iteration, same opcodes), because `store`
checks that its register holds a box where the old opcode indexed the
frame's cells; calling a closure that increments its upvalue +2.1%.
Against #231 everything else is faster.

Two rules the tests taught the code generator: a closure's upvalues stay
live at its write, since `clos` writes the destination before `upval`
reads the boxes; and a captured parameter's box is made after the
`recv`s, which a call storing arguments directly needs to come first.
Boxes at declaration rather than at frame entry came from the conformance
test with 200 captured locals, which otherwise needed 400 registers.

