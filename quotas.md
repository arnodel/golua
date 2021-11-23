# Safe Execution Environments

## Overview

It can be useful to be able to run untrusted code safely. This is why Golua
allows code to be run in a restricted execution environment. This means the following:
- the "amount of CPU" available to the code can be limited
- the "amount of memory" available to the code can be limited
- file IO can be disabled
- the Go interface can be disabled (Golua has a non-standard `golib` builtin package)

### Meaning of limiting CPU
By "amount of CPU" we mean this: the Golua VM periodically emits ticks during
execution.  Not all ticks correspond to the same number of CPU cycles but it is
guaranteed that there is an upper bound to the number of CPU cycles occurring
between two ticks.

Limiting the amount of CPU means declaring that the number of
ticks shouldn't exceed a certain number.

The program is required to terminate before the limit is reached.

### Meaning of limiting memory

By "amount of memory" we mean roughly
- the number of bytes that are allocated on the heap
- the size of the "stack frames" associated with Lua functions and Go functions
  called from Golua.

Memory used can be counted down when it is known that an object is no longer
going to be used (e.g. a Lua function "stack frame"), but in many cases this
does not happen.  So counting memory used works a bit as if GC was mostly turned
off.

Below is an example that currently would run have memory counted as used
increasing linearly in terms of `n`.
```lua
for i = 1, n do
  -- The following creates a new table, consuming memory.  That table will get
  -- GCed shortly but that won't make the amount of memory go down.
  t = {}
end
```

Limiting the amount of memory means declaring that the "amount of memory" used
as defined above shouldn't exceed a certain number.

The program is required to terminate before the limit is reached.

### Disabling IO access and golib

When these restricitions are in place, trying to call a function that perform IO
access (or runs Go code) should return an error, but not terminate the program.

## Safe execution Interface

There are three ways to apply the limits described above.
- When creating the Lua runtime from the program embedding Golua
- Within a Lua program, to safely execute some Lua code
- When starting the standalone `golua` interpreter

The restrictions are managed via the notion of runtime context, which is an
object that accounts for resource limits and resource consumed. A runtime
context is associated with the Lua thread of execution (so there is only one
such context active at a time).
### Standalone golua interpreter

Command line flags allow running the interpreter with restrictions.  Here is the
relevant extract from `golua -help`:
```
  -cpulimit uint
        CPU limit
  -memlimit uint
        memory limit
  -nogolib
        disable Go bridge
  -noio
        disable file IO
```

### Within a Lua program

Golua provides a `runtime` library which exposes two functions

#### `runtime.context()`

Returns an object `ctx` representing the current context.  This object cannot be
mutated but gives useful information about the execution context.

- `ctx.status` is the status of the context as a string, which can be
  `"live"` if this is the currently running context, `"done"` if this execution
  context terminated successfully, or `"killed"` if the context terminated
  because it would otherwise have exceeded its limits.
- `ctx.cpulimit` is the CPU limit for the context.
- `ctx.cpuused` is the amount of CPU used so far in the context (so that will
  change each time for a live context).
- `ctx.memlimit` is the memory limit for the context.
- `ctx.memused` is the amount of memory used so far in the context (so that will
  change each time for a live context).
- `ctx.io` is set to the string `"on"` if IO is enabled, `"off"` otherwise.
- `ctx.golib` is set to the string `"on"` if the Go bridge is enabled, `"off"`
  otherwise.

#### `runtime.callcontext(ctxdef, f, [arg1, ...])`

This function creates a new execution context `ctx` from `ctxdef`, calls
`f(arg1, ...)` in this context, then returns `ctx`. 

By default `ctx` will inherit from the current context: its CPU and memory
limits will be the amount of unused CPU and memory in the current context, and
it inherits the `io` and `golib` flags from the current context.

 The argument `ctxdef` allows restricting `ctx` further.  It is a table with any
of the following attributes.
- `cpulimit`: if set, the CPU limit of `ctx` is set to that number unless it
  would increase it.
- `memlimit`: if set, the memory limit of `ctx` is set to that number unless it
  would increase it.
- `io`: if set to `"off"`, disables IO in `ctx` (has no effect if set to `"on"`).
- `golib`: if set to `"off"`, disables the Go bridge in `ctx` (has no effect if
  set to `"on"`).

Here is a simple example of using this function in the golua repl:
```lua
> ctx = runtime.callcontext({cpulimit=1000}, function() while true do end end)
> print(ctx)
killed
> print(ctx.cpuused, ctx.cpulimit)
999     1000
> print(ctx.io, ctx.golib)
on      on
> print(ctx.memused, ctx.memlimit)
0       nil
```

### When embedding a runtime

There are a `RuntimeContext` interface and a `RuntimeContextDef` in the `runtime` package:

```golang
type RuntimeContext interface {
	CpuLimit() uint64
	CpuUsed() uint64

	MemLimit() uint64
	MemUsed() uint64

	Status() RuntimeContextStatus
	Parent() RuntimeContext

	Flags() RuntimeContextFlags
}

type RuntimeContextDef struct {
	CpuLimit uint64
	MemLimit uint64
	Flags    RuntimeContextFlags
}
```

A Lua runtime of type `*runtime.Runtime` implements the `RuntimeContext`
interface and also has two methods `PushContext(RuntimeContextDef)` and
`PopContext()` that allow managing execution contexts.

```golang
import (
    "os"
    rt "github.com/arnodel/golua/runtime"

)

func main() {
    r := rt.NewRuntime(os.Stdout)

    r.PushContext(rt.RuntimeContextDef{
        MemLimit: 100000,
        CpuLimit: 1000000,
        Flags: rt.RCF_NoIO|rt.RCF_NoGoLib,
    })
    // Now executing Lua code in this runtime will be subject to these limitations
    // If the limits are exceeded, the Go runtime will panic with a
    // rt.QuotaExceededError.

    ctx := r.PopContext()
    // We are back to the initial execution context.  PushContext calls can be
    // nested.  The returned ctx is a RuntimeContext that can be inspected.
}
```

## Random notes

TODOs:
- push Etc: done for requiring mem, should release mem when register is cleared?
- strings: streamline requiring mem

Implementations Guidelines:
- in an unbounded loop require cpu proportional to the number of iterations in
  the loop
- when creating a Value require memory
- when creating a slice of values require memory
- when creating a string require memory
- when calling a Go standard library function you want to require memory /
  cpu depending on the characteristics of this function

Testing guidelines
- write *.quotas.lua test file, using quota.rcall to check that memory and cpu
  are accounted for.
  