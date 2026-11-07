# A control-flow graph for the IR

The passes over the IR (`ir-passes.md`) see a function as a flat list of
instructions.  Within a straight-line stretch they see every read and write;
across a label or a jump they assume nothing.  Several optimisations we want
next need to know more: which code can run after which, what is a loop, where
a loop can be entered.  That is what a control-flow graph gives.  This note
says what the graph would contain, what is particular to golua, and which
passes would use it, in what order.  It is a design note: nothing here is
implemented yet.

## What the passes do without one

- `PropagateCopies` (#174) works stretch by stretch, and forgets what it knows
  at every label and jump.
- `MarkCopyOf` (#197) has the compiler promise that a register is only ever a
  copy of another, because the pass can't see that `advfor` and the loop's
  back jump preserve it.
- The code generator (`ircomp`) reserves a register's slot from its
  `TakeRegister` to its `ReleaseRegister`, in instruction order.  A value
  that must survive a jump back has to stay taken over everything the jump
  can bring execution back to.
- Hoisting constant loads out of loops needs to know where a loop starts and
  ends, and jumps can overlap:

  ```
  A: ...
  B: ...
  C: jump A
  D: jump B
  ```

  A load between B and C is in a loop that runs from A to D: from D
  execution goes back to B, then to C, then back to A.  A register holding
  the hoisted constant must be taken before A and released after D, not
  after C.
- Dead code stays: the body of `while false do ... end` is compiled.

## The graph

**Blocks.**  A block is a run of instructions entered only at its first and
left only after its last.  A block starts at a `DeclareLabel`, or after an
instruction that ends one; it ends after a `Jump`, a `JumpIf` or a tail
`Call`, or just before a `DeclareLabel`.  `TakeRegister` and
`ReleaseRegister` stay where they are: they emit no opcode, but the code
generator needs them.

**Edges.**  A block that doesn't end in a jump falls through to the next one.
A `Jump` goes to its label's block, a `JumpIf` to its label's block and to
the next one, a tail `Call` nowhere: it leaves the function (`return` is a
tail call of the caller's continuation).

**Representation.**  A block is a range of indices into `Instructions`, with
its successors and predecessors as slices of block indices, and a map from
labels to blocks.  The instructions stay a flat list, so the existing passes
and the code generator are unaffected.

## What is derived from it

- **An order**: reverse postorder from the entry.  Blocks it doesn't reach
  are unreachable.
- **Dominators**: block X dominates block Y if every path from the entry to Y
  goes through X.  This is where code can be put so that it has run before Y
  on every path.  The simple iterative algorithm (Cooper, Harvey and
  Kennedy, "A Simple, Fast Dominance Algorithm") is enough at our function
  sizes.
- **Loops, as strongly connected components**: sets of blocks where each can
  reach all the others (Tarjan's algorithm).  For each loop: its blocks, its
  entries (blocks with a predecessor outside the loop) and its exits.  In the
  example above, A to D is one loop: the overlapping jumps are what makes it
  one.
- **Liveness**: at each point, the registers whose value may still be read.
  This is a dataflow analysis over the blocks.  Registers are never reused
  for another purpose in a function, so a bitset over the function's
  registers per block is enough.

## What is particular to golua

**Calls are not edges.**  With continuations, a non-tail `call` comes back to
the next instruction, so it is an ordinary instruction inside a block.  An
error leaves the whole function and needs no edge.  Neither does a
to-be-closed variable's `__close`, which runs during `TruncateCloseStack`
and comes back.

**A loop can have two entries.**  `goto` can jump into a loop at a label
other than the one execution falls into, as long as it doesn't jump into the
scope of a local:

```lua
local x = 0
if c then goto b end
::a::
x = x + 1
::b::
x = x + 10
if x < 50 then goto a end
```

The loop from `a` to the last line is entered by falling into `a` when `c` is
false, and by jumping to `b` when it is true.  Lua and golua both accept it.
Finding loops as strongly connected components, rather than as the "natural
loops" of a single header, handles this.  A pass that wants code before a
loop then needs one place that runs before every entry; if there is none, it
leaves that loop alone.

**Register slots follow instruction order.**  As above and in
`ir-passes.md`, the code generator reserves slots by take and release in
instruction order, not by liveness.  A value that is live where a loop jumps
back needs its register taken from before the loop's first block to after
its last, in instruction order.  Today any pass that makes a value live
longer must arrange that itself; the first user below makes it a property of
the pipeline instead.

**Lines.**  Every instruction carries a line, which line hooks see.  An
inserted instruction gets line 0 ("no line") unless it stands for source
code, so that hooks don't fire on lines that didn't run.

## First user: take and release from liveness

`TakeRegister` and `ReleaseRegister` are register lifetimes written down by
the AST compiler: it knows when it stops needing a temporary, and releases it
at once.  That lets the code generator allocate slots without a liveness
analysis, which was a good trade while there were no passes.  But it fixes
lifetimes before the passes run, and every pass that changes one has to
patch the markers by hand:

- `PropagateCopies` tracks which registers are taken, and only moves a read
  onto a register that is taken at that point;
- `PreloadConstants` (on `register-template`) skips over takes and releases
  to find the instruction after a load;
- hoisting would have to take each hoisted register in the preheader and
  release it after the loop's last instruction, which is the trap of the
  overlapping jumps above.

So the first user of the graph recomputes the markers after the passes:

1. Remove every `TakeRegister` and `ReleaseRegister`.
2. Compute liveness.
3. Give each register an *interval* in instruction order, from its first
   write to the last point where it is live.  If it is live where a loop
   jumps back, the interval covers the whole loop.
4. Take the register at the start of its interval and release it at the end.

`ircomp` is unchanged: it already allocates slots one interval at a time, in
instruction order.  This is linear scan allocation (Poletto and Sarkar,
"Linear Scan Register Allocation").

Points to get right:

- A register that is written but never read still needs a slot at the
  write, or the write could land on a live one: its interval starts at its
  first write even if it is never live.
- A register read on some path before any write is live from the function's
  entry.  That is conservative and correct, provided nothing relies on a
  reused slot being nil; `local x` without a value must load nil.
- A closure holds its cells, and the close stack holds to-be-closed values,
  so their registers can end where this function last reads them.
- golua has no `debug.getlocal`, and error messages don't name locals, so
  nothing sees a local after its last read.

Frames won't get much smaller: the AST compiler already releases temporaries
as soon as they are used, and only named locals that are dead before the end
of their scope free a slot earlier.  The point is what it removes: the first
two constraints in `ir-passes.md` (slots live between take and release;
operands are released just before use) stop applying to passes.

In stages:

1. The pass, with the AST compiler still emitting its markers, which the
   pass discards.  Compare the register counts per function with and without
   it (they should not grow), then run the Lua test corpus and the
   conformance suite.
2. Remove the markers from `astcomp`: large but mechanical.
3. Simplify the passes that tracked them.

## Second user: hoisting constant loads out of loops

For each loop, and each constant loaded inside it into a temporary that
nothing else writes:

1. Give the constant a fresh register for this loop.
2. Load it in a new block before the loop's entry, the *preheader*, and
   redirect the jumps that enter the loop from outside to the preheader.  A
   loop with more than one entry is left alone.
3. Read the fresh register where the temporary was read, and remove the
   load inside the loop.

There is no step for take and release: the first user computes them after
this pass, so the register is taken over the whole loop because it is live
over the whole loop.

In nested loops, the load goes before the outermost loop that contains it
and fits a *window*.  The window bounds the loop's length in instructions and
the number of constants hoisted into one preheader, so that a big function
still gets its inner loops optimised without running out of registers.

The register template (local branch `register-template`) does the same
rewrite of reads, but loads every constant once per call.  The two compare
like this:

| | hoisting | template |
|---|---|---|
| constant loads inside loops (5.05% of executed opcodes) | removed | removed |
| constant loads that run once per call (5.24%) | stay | removed |
| cost per call | none | copying the template |
| registers | only during the loop, bounded by the window | for the whole function |
| runtime, binary chunks, verifier | unchanged | all change |

The template measured −3.6% geomean, mostly on loop-heavy benchmarks.  Only
spectral-norm (−4.2%), most of storage (−2.9%) and part of md5 came from
per-call constants, which hoisting would give up.  The plan is to measure the two
against each other, with a switch in one binary: off, template, hoist.

## Later users, in order

1. **Removing unreachable blocks.**  Small, and it falls out of the order:
   the body of `while false`, code after a `break`, and the jumps the
   compiler emits after a `return` inside an `if` (binary-trees'
   `BottomUpTree` has one after each of its tail calls).
2. **Copy propagation across blocks**, from reaching definitions.  It would
   subsume `MarkCopyOf`.  The move census of 6 Nov 2026 says that what moves
   remain are mostly real data movement, so this is low priority on its own.

## Tests

- One case per shape: straight-line code, `if`/`else`, `while` with the
  entry jump of #196, numeric `for`, nested loops, the overlapping jumps
  above, the two-entry `goto` loop, `break` and `goto` out of a loop,
  unreachable code.
- Random graphs, with the dominators and loops checked against
  brute-force reachability.
- Recomputed take and release: every function of the Lua test corpus and
  the benchmarks compiled both ways, with no more registers than before, and
  the same results.
- Each pass that uses the graph gets its own tests, plus the Lua test
  corpus, the conformance suite and an executed-opcode count, as in
  `ir-passes.md`.

## Size

About 100 lines for blocks and edges, 60 for the order and dominators, 50 for
loops, 60 for liveness and 50 for the intervals, built per function.
