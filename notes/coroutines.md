# Coroutines without goroutines

This note records where coroutines stand, why they could be much cheaper, what
stands in the way, the approaches that were considered and the one chosen.
Nothing described here is merged yet, except where a PR number is given.  The
measurements are from 14 and 15 November 2026, on the tip of `lua5.5` at that
time (41270d1), built with the profile and without quotas, on an Apple M1 Max,
against Lua 5.5.1.

## Where we are

A coroutine is a goroutine (`runtime/thread.go`).  `Thread.Start` starts one
that waits on a channel; `Resume` and `Yield` hand values over that channel in
each direction, under a pair of mutexes, and block the side that is not
running.  Only one goroutine runs at a time, so this is a hand-off, never
concurrency.

This was the simplest way to give each coroutine what it needs most: its own
Go stack.  A coroutine can be suspended while Go functions of its own are
running (a `pcall`, a metamethod called from Go, the comparison function of
`table.sort`), and a goroutine keeps those frames for as long as it takes.

It costs three things.

| | golua | Lua 5.5.1 |
|---|---|---|
| resume + yield, round trip | 635 ns | 110 ns |
| the same through `coroutine.wrap` | 612 ns | 95 ns |
| the same with one processor (`GOMAXPROCS=1`) | 345 ns | |
| create a coroutine and run it to its end | 4630 ns | 265 ns |
| create one and never resume it | 3880 ns | 181 ns |
| `coroutine-ring` benchmark, one run | 78 ms | 11.6 ms |

- **Switching** costs six times the reference.  About half of it is the Go
  scheduler passing the work from one processor to another, which also leaves
  the coroutine's data cold in the cache: a yield from five Lua calls deep
  costs 1435 ns, where the five calls and returns account for about 330.
- **Creating** costs twenty times the reference: a goroutine and its stack.
- **A coroutine dropped while suspended is never freed.**  Its goroutine waits
  on its channel for ever and holds the thread and everything the coroutine
  can reach.  Leaving a `for ... in coroutine.wrap(...)` loop with `break` is
  enough.  200,000 generators abandoned after their first value, each holding
  1 KB, peak at 1425 MB; the reference stays at 1 MB.

`coroutine-ring` is the outlier of the benchmark suite at 6.8 times the
reference, where the suite as a whole is under 2.

## What a coroutine is in this VM

golua runs Lua in continuation-passing style.  A function's frame is a
continuation on the heap, and `Thread.RunContinuation` is a loop that runs one
continuation, gets the next one from it, and goes on:

```go
for c != nil {
    t.currentCont = c
    next, err = c.RunInThread(t)
    ...
    c = next
}
```

The Lua "stack" is the chain of continuations, each holding the one it returns
to.  There is no Go stack frame per Lua call.

So a suspended coroutine whose frames are all Lua needs no stack of its own:
it is one continuation (the one waiting for the values of the next resume)
plus the thread's close stack.  Resuming it is running that continuation;
yielding is remembering the continuation that waits and going on with the
resumer's.  The loop can do both by changing which thread it is running.

### The prototype

A throwaway prototype did exactly this (about 170 lines in
`runtime/thread.go`, with the coroutine library switched over to it).

- A *light* thread has no goroutine.  It holds the function to run until its
  first resume, then the continuation to resume.
- `coroutine.resume`, the function `coroutine.wrap` returns, and
  `coroutine.yield` are Go functions, called by the loop like any other.  They
  push the values into the other side's continuation, leave the thread and
  continuation to go on with in the runtime, and return a sentinel error.
- The loop keeps the thread it is running in a local variable.  On the
  sentinel it replaces thread and continuation and carries on.  The sentinel
  travels on the error path, which is already off the fast path, so the loop
  gets no new test for the ordinary case.
- When a light coroutine's chain of continuations ends, or it dies of an
  error, the loop closes its pending to-be-closed values and hands the values
  or the error to whoever resumed it.

| | today | prototype | Lua 5.5.1 |
|---|---|---|---|
| resume + yield | 635 ns | 111 ns | 110 ns |
| through `wrap` | 612 ns | 97 ns | 95 ns |
| as a `for ... in` iterator | 660 ns | 114 ns | 96 ns |
| create and run to the end | 4630 ns | 257 ns | 265 ns |
| create, never resumed | 3880 ns | 103 ns | 181 ns |
| `coroutine-ring`, one run | 78 ms | 12.5 ms | 11.6 ms |
| 200,000 abandoned generators, peak | 1425 MB | 15 MB | 1 MB |

Coroutines are level with the reference, and a dropped coroutine is ordinary
garbage.  On the suite the cycle count falls by 6.5% (geometric mean of 25),
all of it from `coroutine-ring`.  The other benchmarks execute 0.4 to 0.7%
more instructions where they make many calls; their cycles move by 0.0 to
0.5%, which is inside the noise of code placement.  The cause was not found
and a real version should find it.

The prototype handles one case only, and refuses the rest.

## The obstacle: Go frames between the resume and the yield

A yield can be asked for while a Go function of the coroutine is on the Go
stack, having called back into Lua through a nested `RunContinuation`:

```lua
local co = coroutine.wrap(function()
    pcall(function()
        coroutine.yield(1)     -- pcall's Go frame is on the stack here
    end)
end)
```

The frame of `pcall` cannot be turned into a continuation: it is a Go function
in the middle of its work.  The prototype refuses such a yield ("attempt to
yield across a Go-call boundary").

Where a yield is allowed, in the reference and in golua today:

| the yield is inside | Lua 5.5.1 | golua today |
|---|---|---|
| a Lua function called by Lua code | yes | yes |
| `pcall`, `xpcall` | yes | yes |
| a metamethod called by an operator or an index | yes | yes |
| `__call`, a `for ... in` iterator | yes | yes |
| a `__close` handler at the end of a block | yes | yes |
| `__pairs` called by `pairs` | yes | yes |
| the comparison function of `table.sort` | no | yes |
| the replacement function of `string.gsub` | no | yes |
| `__tostring` called by `tostring` | no | yes |
| `__newindex` called by `table.insert` | no | yes |
| a reader function given to `load` | no | yes |
| a debug hook | no | yes |
| a `__close` handler run by `coroutine.close` | no ("attempt to yield across a C-call boundary") | hangs |

The reference refuses a yield across a C function that was not written to be
suspended (the manual, §4.5).  golua allows it everywhere, because a goroutine
can be suspended anywhere: this is GOLUA-013 in `CONFORMANCE.md`.  It matters
to a Go program embedding golua: a Go function that calls a Lua callback which
yields just works, where the C API asks for a continuation function written by
hand.  The last line is a bug of today's implementation, not yet fixed.

How often does a yield have a Go frame in the way?  An instrumented build
logged every yield:

| | yields | with a Go frame in the way |
|---|---|---|
| golua's own tests | 100,029 | 2, both through `pcall` |
| the conformance suite | 1357 | 95 |

Of the 95, 75 are inside metamethods and `__close` handlers, 19 through
`pcall` or `xpcall`, 1 through `__pairs`.  The suite tests these on purpose,
so the proportion says little about programs.  What it does say is that the
kinds of Go frame are few: `pcall` and `xpcall`, metamethods called by the run
loop, `__close` handlers.  In real programs `pcall` is the one to expect: a
scheduler usually runs each task inside one.

### Why the choice cannot be made at the yield

The obvious hybrid is "run light, and fall back to a goroutine when a yield
finds a Go frame in the way".  It does not work, and the reason shapes
everything else.

A light coroutine runs in the loop of whoever resumed it, on the resumer's
goroutine.  Take a host program that calls a Lua function on the main thread:
that is a loop, L.  The function resumes a light coroutine, which runs in L.
The coroutine calls `pcall`, whose Go frame P goes on the stack above L and
runs a nested loop.  Inside it the coroutine yields.

To suspend the coroutine, P must be kept: it is the coroutine's unfinished
business.  The main thread must go on, and when its function returns, L must
return to the host program.  But L is under P on the same Go stack, and a Go
function cannot return while another sits above it.  Running the main thread
on another goroutine does not help: the host program is waiting on this one.

More generally, a goroutine's stack can hold the Go frames of one thread only,
since threads are suspended and resumed in any order and a stack is not.  By
the time a light coroutine has a Go frame that called back into Lua, that
frame is on somebody else's stack.  So the coroutine has to be on a goroutine
of its own **before** such a Go frame is made.

## The approaches

**1. Make the Go frames go away.**  `pcall`, `xpcall`, the metamethod calls of
the run loop and `__close` become continuation-based: instead of running a
nested loop they make a continuation and return to the loop they are in, as
`__call` already does.  An error then looks for its handler along the chain of
continuations.  What is left (callbacks of library functions such as
`table.sort`) refuses a yield, as the reference does.

- One mechanism.  The goroutine code, the mutexes and the channel go.
- Every coroutine is light: all of the speed, and no leak at all.
- Gives up GOLUA-013, and with it the Go function that calls a Lua callback
  that yields.
- A lot has to change before the first coroutine can be light without
  breaking something: error handling, quota contexts (`CallContext`), every
  operator's metamethod path.

**2. A hybrid: light until a goroutine is needed.**  A coroutine starts light.
Before one of its Go frames can call back into Lua, it is moved to a
goroutine, and from then on it is today's coroutine.

- Keeps everything golua allows today, GOLUA-013 included.
- Generators, iterators and coroutines whose frames are all Lua get the whole
  gain.  A coroutine that uses `pcall` or a function metamethod is moved, and
  keeps today's cost and today's leak.
- Two mechanisms to maintain, and the move between them.
- Can be done in steps that each leave everything working.

**The hybrid is the one chosen**, for the contexts where a yield across a Go
function is useful.  The two are not exclusive: the first approach, applied to
`pcall` alone, is a later step of the second that keeps more coroutines light.

## The hybrid

### Moving a coroutine to a goroutine

The loop that is running a light coroutine starts a goroutine for it, whose
own loop begins with the continuation that was about to run, and then waits
for the coroutine to yield or finish, exactly as a resumer waits today.  From
then on the coroutine is resumed and yields through the channel, with the
existing code.  The move is one way.

The mixtures work out, because a light coroutine has no ties: it runs in
whatever loop resumes it.  A light coroutine resumed by a coroutine that has a
goroutine runs in that one's loop; one that was suspended can be resumed later
from another goroutine; if it needs a goroutine itself, its host's loop waits
for it like any resumer.

There are two places where a coroutine can come to need a goroutine.

### Go functions: a declaration

When the next continuation of a light coroutine is a Go function that may call
back into Lua, the coroutine is moved before the function runs.  Which Go
functions may is a declaration made when the function is registered, next to
the safety flags that `SolemnlyDeclareCompliance` already carries.

The declaration should say "this function never calls Lua", and a function
without it is assumed to.  The standard library is ours to declare.  A
function of an embedding program that says nothing then makes a coroutine
slower, never wrong; the other way round, a forgotten tag would turn a yield
into an error.  A function that declares it never calls Lua and does is a bug
in the function, to report as an error when it happens.

Care is needed with what "calls Lua" covers: `tostring`, `print` and
`string.format` can call `__tostring`; the `table` functions honour `__index`
and `__newindex`; `pairs` calls `__pairs`.  `coroutine.resume`, `wrap` and
`yield` themselves return to the loop and never nest.

### Metamethods: run the opcode again

A metamethod called by an operator or an index has no Go function to declare:
the run loop calls it in the middle of an opcode.  But every call from Go into
Lua goes through one function, `Thread.RunContinuation`.  When a light
coroutine arrives there, the call can be refused before the callee has run a
single opcode, with a sentinel error.  The opcode returns it like any error,
having saved its position; the loop moves the coroutine to a goroutine and
runs the same continuation again, which runs the opcode again, and this time
the call goes through.

This is sound if the opcode has done nothing that shows before the call it
makes.  An audit checked it two ways: by reading every opcode that can reach a
metamethod, and with a harness that refuses, once per run of a Lua function,
the first call into Lua made from an opcode, and runs the function again.

| the harness with | conformance suite | golua's tests |
|---|---|---|
| no fix | fails | fails |
| the three fixes below, debug hooks exempt | passes, 3149 refusals | passes, 157,495 refusals |

Opcodes that can be run again as they are:

- arithmetic, bitwise operators, comparisons (both the ones that give a value
  and the ones that decide a jump), concatenation, unary minus, bitwise not,
  length;
- index load and store, by register and by constant key, and the check that a
  global is not already defined;
- `__call` makes no nested call at all;
- none of the helpers between the opcode and the call rewrites or swallows the
  error, so the sentinel reaches the loop.

Places that need a change:

1. **Closing a block** (`cltrunc`).  `cleanupCloseStack` takes a value off the
   close stack before calling its `__close` handler; run again, the opcode
   would never call it.  The question must be asked before the value is taken
   off.
2. **A return or a tail call with values to close.**  The call opcode has
   pushed its arguments, moved its position and cleared the callee's register
   before it closes anything.  It must test for pending values first, at the
   top.  The test can take the place of the call to `cleanupCloseStack` that
   every return makes today, so it should cost nothing.
3. **The line hook.**  Its error path returns without saving the position.
   One line.
4. **Call and return hooks.**  The result of calling the hook is thrown away,
   so a refusal would be lost, and the opcode cannot be run again at that
   point anyway.  A coroutine with a debug hook set is given a goroutine from
   the start.

Two cases are outside the run loop: a coroutine that dies with values still to
close, and `coroutine.close`.  Their handlers are called from Go.  They can
run in a nested loop with a yield refused, which is what the reference does
for `coroutine.close`; what it does for a dying coroutine is still to check.

### What the hybrid leaves as it is

- A coroutine that calls `pcall` is moved to a goroutine at the first one.
- A coroutine on a goroutine still costs what it costs today, and is still
  never freed if it is dropped while suspended.
- `coroutine.close` hangs when a `__close` handler yields.

## The plan

1. **Light coroutines, the move, the declaration, and the four changes
   above.**  Everything keeps working, yields inside metamethods and library
   callbacks included.  Generators and iterators get the gain.
2. **`pcall` and `xpcall` without a nested loop**, so that a coroutine using
   them stays light.  This can be measured by itself: it also takes a Go frame
   and a nested loop out of every protected call.
3. **The leak of coroutines that have a goroutine**, and the hang in
   `coroutine.close`.

Open questions for the first step:

- Quota contexts end a computation with a panic that unwinds the Go stack.  A
  light coroutine is on its resumer's stack, so the panic passes through; its
  status has to be left right (the prototype leaves it "normal", where it
  should be "dead").
- `debug.getinfo` and `debug.traceback` of a suspended light coroutine: the
  continuation of `coroutine.yield` goes back to its pool at the switch, so
  the thread must remember where it stopped some other way.
- The call hook at a coroutine's first resume.
- The 0.4 to 0.7% more instructions seen on call-heavy benchmarks.

## What was built

Step 1 is in two parts, and both differ from the plan above in ways worth
knowing before reading the code.  The code's own map is the comment at the top
of `runtime/lightthread.go`.

- **The declaration is about yields, not about calling Lua.**  A Go function
  declared `NoYieldAcross` may call Lua, and that Lua code cannot yield: it
  gets the error Lua gives for a yield across a C function.  The whole
  standard library is declared but for `dofile`, which with `pcall` and
  `xpcall` (below) matches where Lua allows a yield and where it does not
  (`__pairs` is called as a tail call, so it needs no exception).  So the
  sentence in step 1,
  "yields inside library callbacks included", no longer holds: a yield in a
  `table.sort` comparison or a `string.gsub` replacement is an error, where
  golua used to allow it.  A Go function of an embedding program keeps the
  choice: undeclared, it can be yielded across and costs the coroutine its
  goroutine.
- **A debug hook cannot yield either**, so a hook is no reason for a
  goroutine, and the hook events of a light coroutine are those of any other.
- **While Lua code that cannot yield is running, a light coroutine can do
  anything**: nothing there can be left waiting on the Go stack by a yield.
- **`pcall` and `xpcall` have no Go frame** (step 2, in
  `runtime/protectedcall.go`).  The function they call returns to a
  continuation that stays in the chain while it runs and marks where an error
  stops; the loop that sees an error looks for it down the chain.  So a
  coroutine that uses them keeps having no goroutine, and can yield inside
  them.  The message handler of `xpcall` is found the same way, which also
  means it is the handler of the `xpcall` the error is in: it used to be the
  handler of whichever `xpcall` had started last, in any coroutine.
- **A light coroutine stops and goes on by running its current continuation
  again**: after a yield it is the call of `yield`, which returns the values
  of the resume the second time.  That answers the open questions about what
  a suspended light coroutine remembers.

## Found on the way

Each call from Go into Lua left its result holder reachable from the thread,
and the next one chained onto it: a loop going through a function metamethod
kept every one of them alive until the function returned, and calls made by a
host program were never freed.  Fixed separately (#271).
