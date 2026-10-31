# Writing an optimisation pass over the IR

Golua compiles Lua in three steps: AST (`ast`, `parsing`) to IR (`astcomp`,
`ir`), IR to opcodes (`ircomp`, `code`), and the runtime executes the opcodes
(`runtime/luacont.go`).  Optimisation passes run on the IR, between the first
two steps, in `runtime/lib.go` (`compileLuaStat`).  The first such pass is
`ir.PropagateCopies` (`ir/copyprop.go`, PR #174); this note records what it
had to learn so that the next pass does not have to learn it again.

## Why passes pay

The run loop's cost is mostly per opcode, not per operation: removing one
plain register move saves about as much time as removing an arithmetic
opcode.  Measured on fannkuch-redux, −19.7% opcodes executed gave −18.8% time
although the removed opcodes were all moves.  So the first question for any
candidate is "how many executed opcodes does it remove", not "how expensive
are they".

## Measure what executes first

Static counts over a disassembly mislead: a pattern that is common in source
may be rare in hot loops and vice versa.  Count what the benchmarks execute:

1. Copy the repo somewhere disposable (`git archive HEAD | tar -x -C dir`).
2. In `RunInThread`, after `opcode := opcodes[pc]`, call a counter that
   classifies the opcode into a shape (move, loadint, indexK, jumpif, ...)
   and counts shapes and adjacent pairs of shapes.
3. Add a test to `benchmarks` that runs each benchmark once (`newGolua`,
   `runGolua`) and dumps the counts, discarding the setup run.
4. Compare totals per benchmark between builds.

The pairs are what point at a rewrite: `move > setindexK` said "the table is
copied before every field store", `binop > move` said "results go to a
temporary and are then moved into the local", `not > jumpif` says "a negated
condition costs an opcode".

## The IR model, as a pass sees it

- A function is a flat list of `ir.Instruction` with parallel `Lines`.  Every
  instruction implements `Operands`: `Reads()`, `Writes()`,
  `ReplaceRead(from, to)`.  Keep the two lists honest: a table being stored
  into is read; `prepfor`/`advfor` both read and write their registers;
  `MkVarargTable` writes its etc register too.
- Registers are never reused for another purpose within a function
  (`GetFreeRegister` always appends), so a register means one thing
  throughout and counts of its reads and writes over the whole function are
  exact.  `RegData` says whether a register is a cell, a named local
  (`IsLocal`), a compile-time constant.
- Control flow: `DeclareLabel` marks a join point, `Jump`/`JumpIf`/tail
  `Call` end a straight-line stretch.  Within a stretch every read and write
  is visible; across stretches assume nothing unless you do real dataflow.
- `TakeRegister`/`ReleaseRegister` are lifetime markers emitted when a
  register's reference count goes 0→1 and 1→0.  They emit no opcode.

## Constraints that come from the code generator

These are not Lua semantics; they are how `ircomp` turns IR registers into
the frame's code registers, and a pass that ignores them produces code that
reads garbage.  Each one was found by a failing Lua test.

1. **Slots live between take and release.**  `ircomp` gives an IR register a
   code slot on first mention and reserves it only while the register is
   taken; a register that is never taken keeps its slot only until the next
   register is first used.  So a pass may extend a *read* of a register only
   if the register is taken at the point of the new read.  Example: after
   `x, y = f()` the results arrive in untaken temporaries; reading them later
   instead of `x` and `y` reads recycled slots.
2. **Operands are released just before use.**  The compiler emits
   `release r12; release r13; r12[r13] <- r6`.  That is fine for it (nothing
   is allocated in between), and a pass must not treat the release of a
   register as the end of the *value* it holds, only as the end of its slot
   reservation.
3. **Cells are shared by reference.**  A local captured by a closure, or an
   upvalue, lives in a cell; a function called between two instructions can
   change it.  Never substitute a cell for anything or anything for a cell.
4. **Read-write operands keep their name.**  `prepfor`/`advfor` update their
   registers in place; substituting the read side sends the write elsewhere.
5. **Named locals stay observable.**  Deleting an assignment to a named local
   changes what `debug.getlocal` and error messages see.  Delete only
   assignments to temporaries.

## Lua semantics to preserve

- In a multiple assignment all right-hand sides are evaluated before any
  assignment, and `i, a[i] = i+1, 20` stores into the *old* `a[i]`.
  `runtime/lua/assign.lua` has the cases.
- The order in which a statement's sub-expressions are evaluated is
  undefined by the manual but fixed by the IR; a pass must not reorder
  instructions with side effects, only change which register they read.
- Quotas: fewer opcodes means less CPU charged.  Tests that assert a quota
  is hit at a given iteration count (`runtime/lua/thread.quotas.lua`) may
  need their counts retuned, which is legitimate.

## How to test a pass

- Unit tests on hand-built IR (`ir/copyprop_test.go`): one case per rule,
  each a small instruction list and the expected result.
- The Lua test corpus (`go test ./...`, both builds) and the conformance
  suite (`golua-tests`) with the pass on.  These found every constraint
  above.
- Disassemble a small program with `golua -dis` before and after and put
  the sequences in the PR and the code comments; they are the clearest
  description of what a pass does.
- Measure: opcode counts per benchmark, then the interleaved A/B
  (`benchmarks/README.md`).  Expect time to track opcodes on tight loops
  and to lag them on call-heavy code.

## Candidates from the counts (31 Oct 2026)

- `not` then `jumpif` on a temporary: one `jumpif` with the sense flipped
  (3–4% of richards' and fannkuch's opcodes).
- `x, y = f()`: `recv` into two temporaries then two moves; `recv` could
  write the locals directly (needs a multi-destination form of the
  "write straight to the destination" rewrite).
- Small integer literals reloaded before each use (15% of fannkuch): an
  immediate operand form of arithmetic and indexing is an opcode-level
  change, not a pass.
- The call protocol (`cont`, `push`..., `call`) is 25–35% of the opcodes in
  richards, deltablue, towers and json; a structural change.
- `ir/fold.go`'s pairwise `FoldMoveReg` is unsafe (no liveness) and is
  superseded; `FoldTakeRelease` saves no opcodes.  Both can go.
