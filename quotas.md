# Safe Execution Environments

- [Safe Execution Environments](#safe-execution-environments)
  - [Overview](#overview)
    - [Meaning of limiting CPU](#meaning-of-limiting-cpu)
    - [Meaning of limiting memory](#meaning-of-limiting-memory)
    - [Disabling IO access and golib](#disabling-io-access-and-golib)
  - [Safe Execution Interface](#safe-execution-interface)
    - [In the standalone golua interpreter](#in-the-standalone-golua-interpreter)
    - [Within a Lua program](#within-a-lua-program)
      - [`runtime.context()`](#runtimecontext)
      - [`runtime.callcontext(ctxdef, f, [arg1, ...])`](#runtimecallcontextctxdef-f-arg1-)
    - [When embedding a runtime in Go](#when-embedding-a-runtime-in-go)
      - [`(*Runtime).PushContext(RuntimeContextDef)`](#runtimepushcontextruntimecontextdef)
      - [`(*Runtime).PopContext() RuntimeContext`](#runtimepopcontext-runtimecontext)
      - [`(*Runtime).CallInContext(def RuntimeContextDef, f func()) RuntimeContext`](#runtimecallincontextdef-runtimecontextdef-f-func-runtimecontext)
  - [How to implement resource limits](#how-to-implement-resource-limits)
    - [CPU limits](#cpu-limits)
      - [`(*Runtime).RequireCPU(n uint64)`](#runtimerequirecpun-uint64)
    - [Memory limits](#memory-limits)
      - [`(*Runtime).RequireMem(n uint64)`](#runtimerequirememn-uint64)
      - [`(*Runtime).ReleaseMem(n uint64)`](#runtimereleasememn-uint64)
      - [`(*Runtime).RequireBytes(n int) uint64`](#runtimerequirebytesn-int-uint64)
      - [`(*Runtime).RequireSize(sz uintptr) uint64`](#runtimerequiresizesz-uintptr-uint64)
      - [`(*Runtime).RequireArrSize(sz uintptr, n int) uint64`](#runtimerequirearrsizesz-uintptr-n-int-uint64)
      - [`(*Runtime).ReleaseBytes(n int)`](#runtimereleasebytesn-int)
      - [`(*Runtime).ReleaseSize(sz uintptr)`](#runtimereleasesizesz-uintptr)
      - [`(*Runtime).ReleaseArrSize(sz uintptr, n int)`](#runtimereleasearrsizesz-uintptr-n-int)
    - [Restricting access to library functions](#restricting-access-to-library-functions)
  - [Random notes](#random-notes)
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

## Safe Execution Interface

There are three ways to apply the limits described above.
- When creating the Lua runtime from the program embedding Golua
- Within a Lua program, to safely execute some Lua code
- When starting the standalone `golua` interpreter

The restrictions are managed via the notion of runtime context, which is an
object that accounts for resource limits and resource consumed. A runtime
context is associated with the Lua thread of execution (so there is only one
such context active at a time).
### In the standalone golua interpreter

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

### When embedding a runtime in Go

There is a `RuntimeContext` interface in the `runtime` package.  It is
implemented by `*runtime.Runtime` and allows inspection of the current execution
context.  We will see further down that contexts that are terminated are also
available via this interface.

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
```

The `runtime` package also defines a `RuntimeContextDef` type whose purpose is
to specify the properties of a new execution context to create.

```golang
type RuntimeContextDef struct {
	CpuLimit uint64
	MemLimit uint64
	Flags    RuntimeContextFlags
}
```

As mentioned above, a Lua runtime is of type `*runtime.Runtime` and implements the `RuntimeContext`
interface.  It also implements two methods.

#### `(*Runtime).PushContext(RuntimeContextDef)`

Creates a new context from the definition and makes it the active context.  As
described in the Lua section, the new context is not allowed to be less
restrictive than the one it replaces.

#### `(*Runtime).PopContext() RuntimeContext`

Removes the active context from the "context stack" and returns it.  It ensures
that resources consumed in the popped context will be accounted for in the
parent context.

Here is a simple example of how they could be used.

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

    // Do something in this context

    ctx := r.PopContext()
    // We are back to the initial execution context.  PushContext calls can be
    // nested.  The returned ctx is a RuntimeContext that can be inspected.
}
```

The `*runtime.Runtime` type has another method.

#### `(*Runtime).CallInContext(def RuntimeContextDef, f func()) RuntimeContext`

Similar to Lua's `runtime.callcontext`.  It is a convenience function to run
some code in a given context, catching the `QuotaExceededError` panics if they
occur and returning the finished context. So the above could be rewritten safely
as follows.

```golang
import (
    "os"
    rt "github.com/arnodel/golua/runtime"

)

func main() {
    r := rt.NewRuntime(os.Stdout)

    ctx := r.CallInContext(rt.RuntimeContextDef{
        MemLimit: 100000,
        CpuLimit: 1000000,
        Flags: rt.RCF_NoIO|rt.RCF_NoGoLib,
    }, func() {
        // Do something in this context
    })

    // Panics due to quota exceeded will be recovered from.
}
```

## How to implement resource limits


### CPU limits

The basic means of enforcing CPU limits is the following.
#### `(*Runtime).RequireCPU(n uint64)`

This method checks that `n` units of CPU are available.  If that is the case,
the amount of CPU used is updated and execution continues.  Otherwise, the Go
thread panics with `runtime.QuotaExceededError`.

The approach is to call `RequireCPU` before a unit of work is done.
- In a loop an amount of CPU should be required that is proportional to the
  number of iterations.
- Nested Go function calls should require CPU proportional to the depth of the
  nested calls.
- When running code in third party packages (including the Go Standard Library)
  it should be possible to obtain and upper bound to the amount of CPU required
  ahead of the call and require it.  If the third party function is given a
  callback it may be possible to use that (e.g. `sort.Sort`).

### Memory limits

The basic means of enforcing memory limits are the following.  

#### `(*Runtime).RequireMem(n uint64)`

This methods checks that `n` units of memory are available.  If that is the case,
the amount of CPU used is updated and execution continues.  Otherwise, the Go
thread panics with `runtime.QuotaExceededError`.

#### `(*Runtime).ReleaseMem(n uint64)`

This methods reduces the amount of memory used by `n` units (if possible).  It
is generally not used but can be useful in some cases (e.g. when a big temporary
object needs to be allocated).

Often we know how much memory is required in terms of bytes or size of data
structures, so there are some convenience methods to address that.

#### `(*Runtime).RequireBytes(n int) uint64`

Require enough memory to store `n` bytes.  Return the number of memory units
required.

#### `(*Runtime).RequireSize(sz uintptr) uint64`

Require enough memory to store an obect of size `sz`, size as returned by
`unsafe.Sizeof()`.  Return the number of memory units required.

#### `(*Runtime).RequireArrSize(sz uintptr, n int) uint64`

Require enough memory to store `n` objects of size `sz`, e.g. a slice or an
array of objects.  Return the number of memory units required.


There are corresponding methods for releasing memory

#### `(*Runtime).ReleaseBytes(n int)`

#### `(*Runtime).ReleaseSize(sz uintptr)`

#### `(*Runtime).ReleaseArrSize(sz uintptr, n int)`

The approach is to call `RequireMem` or one of the derived method before some
memory allocation.  Memory allocation occurs when
- A new string is created
- A new table is created
- A new item is inserted into a table
- A new Lua closure is created
- A new Lua continuation is created (that is akin to a "Lua call frame")
- A new Go function is created
- A new UserData instance is created
- Buffered IO occurs
- Lua source code is compiled

Moreover it may be that calling a function in the standard library can cause
memory allocations.

In some case it may be appropriate to return memory.  An example is when a Lua
continuation ends.  Returning its memory allows tail-calls to have the same
memory footprint as loops.

### Restricting access to library functions

TODO
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
  