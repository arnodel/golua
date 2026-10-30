# The run loop and what it costs per instruction

Notes from measuring `(*LuaCont).RunInThread` (October 2026, Go 1.25,
M1 Max, `-tags noquotas` builds unless stated).

## Numbers

- A bare `for i = 1, N do end` loop runs at about 16 ns per iteration,
  roughly 8 ns (25 cycles) per opcode.  That is the per-instruction
  overhead of the VM, before any opcode does useful work.
- The line hook test at the top of the loop accounts for about 0.8 ns
  of that.  Deleting the test altogether (not an option, it is how line
  hooks work) is worth 5.5% geomean on the benchmark suite, so that is
  the ceiling for anything done to the loop header.

## How the compiler treats the loop

`RunInThread` compiles to about 14 KB of code and most opcodes can call
out (metamethods, errors, hooks).  Go keeps the loop's variables (`pc`,
`regs`, `cells`, `opcodes`, `consts`, `c`, `t`) on the stack and loads
them into registers where they are used.  On the back edge from each
opcode to the top of the loop, it reloads whatever the top expects to
find in registers.  This is fine: each opcode pays only for what it
clobbered.

What is not fine is a *join* on the common path where one of the
incoming paths comes from a call.  At a join the compiler picks one
register assignment that every incoming path must satisfy; after a
call, all registers are dead; so if the common path and a post-call
path meet before the opcode fetch, the common path pays to reload the
whole loop state from the stack on every instruction.  Measured, that
was eleven reloads per opcode.

### Rule

Whatever sits at the top of the loop, before the opcode fetch, is paid
on every opcode.  Keep it to register-cheap, call-free tests, and make
sure any path that may call out leaves the loop (or goes back to the
top) rather than rejoining the common path.

## The line hook (applied)

The hook test used to be

    if hooks enabled { if new line { lastLine = line; call hook } }
    opcode := opcodes[pc]

so the post-call path flowed straight into the fetch.  Now the hook
path ends in `continue RunLoop`: run the hook, then start the
instruction over (the second time round the line matches `lastLine` and
nothing is called).  `RequireCPU` is charged after the test so the
instruction is counted once.  `lastLine` became a field of `LuaCont` so
the loop carries no hook state of its own.  Worth about 2% geomean;
neither half of the change measures anything on its own.

Things that were tried and measured as no change: caching the hook flag
in a local before the loop (slower: one more stack slot to reload),
making the two-bit flag test a one-bit test, and removing the nil-check
loads that `t.RequireCPU(1)` leaves behind in the noquotas build.

## Ideas not yet tried

- Treat a hook as a suspension point: save `pc` and return `c` to the
  thread loop, which re-enters `RunInThread` for the next instruction.
  Says the same thing as the `continue` in continuation terms and
  rejoins nothing.
- The real headroom is the 25 cycles per opcode, most of which is
  register access (`getReg`/`setReg` go through a cell check and two
  bounds checks) and stack traffic for the loop state.  Anything that
  shrinks the loop's live state or the per-register work moves every
  benchmark.
