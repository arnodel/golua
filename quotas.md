# Safe Execution Environments (alpha)

- [Safe Execution Environments (alpha)](#safe-execution-environments-alpha)
  - [Overview](#overview)
    - [Meaning of limiting CPU](#meaning-of-limiting-cpu)
    - [Meaning of limiting memory](#meaning-of-limiting-memory)
    - [Other restrictions](#other-restrictions)
    - [Restricting file access](#restricting-file-access)
    - [Working directory](#working-directory)
  - [Safe Execution Interface](#safe-execution-interface)
    - [In the standalone golua interpreter](#in-the-standalone-golua-interpreter)
    - [Within a Lua program](#within-a-lua-program)
      - [`runtime.context()`](#runtimecontext)
      - [`runtime.callcontext(ctxdef, f, [arg1, ...])`](#runtimecallcontextctxdef-f-arg1-)
      - [`runtime.killcontext()`](#runtimekillcontext)
      - [`runtime.contextdue()`](#runtimecontextdue)
      - [`runtime.stopcontext()`](#runtimestopcontext)
    - [When embedding a runtime in Go](#when-embedding-a-runtime-in-go)
      - [`(*Runtime).PushContext(RuntimeContextDef)`](#runtimepushcontextruntimecontextdef)
      - [`(*Runtime).PopContext() RuntimeContext`](#runtimepopcontext-runtimecontext)
      - [`(*Runtime).CallContext(def RuntimeContextDef, f func() *Error) (RuntimeContext, *Error)`](#runtimecallcontextdef-runtimecontextdef-f-func-error-runtimecontext-error)
      - [`(*Runtime).TerminateContext(format string, args ...interface{})`](#runtimeterminatecontextformat-string-args-interface)
  - [Finalizers and runtime contexts](#finalizers-and-runtime-contexts)
  - [How to implement the safe execution environment](#how-to-implement-the-safe-execution-environment)
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
    - [Restricting access to Go functions.](#restricting-access-to-go-functions)
      - [`ComplianceFlags`](#complianceflags)
      - [`(*GoFunction).SolemnlyDeclareCompliance(ComplianceFlags)`](#gofunctionsolemnlydeclarecompliancecomplianceflags)
## Overview

First of all: everything in this document is subject to change!

It can be useful to be able to run untrusted code safely. This is why Golua
allows code to be run in a restricted execution environment. This means the following:
- the "amount of CPU" available to the code can be limited
- the "amount of memory" available to the code can be limited
- file IO can be disabled, or restricted to some directories
- unsafe Go functions accessible via modules can be disabled

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

### Other restrictions

When these restricitions are in place, trying to call a function that perform IO
access (or runs unsafe) should return an error, but not terminate the program.

### Restricting file access

Rather than disabling file IO altogether, a context can be given access to some
directories only.  Each directory is granted a set of actions:

- `r`: read a file
- `w`: write to an existing file
- `c`: create a file
- `d`: delete or rename a file
- `*`: all of the above

Files outside the granted directories cannot be accessed at all.  This is
enforced using Go's [`os.Root`](https://pkg.go.dev/os#Root), so a file cannot
escape a granted directory via `..`, symbolic links or alternative spellings of
its path.  Note that hard links and mount points inside a granted directory are
not detected.

The rules are as follows.
- An action on a file is allowed if any granted directory containing the file
  allows it.  So a grant cannot restrict access to part of another granted
  directory.  E.g. `{dir=".", allowed="r"}, {dir="out", allowed="*"}` allows
  reading anything in the current directory but writing only in `out`.
- Relative paths are resolved against the context's [working
  directory](#working-directory) first, then checked against the grants.
- Opening an existing file with a mode that may create it (e.g. `"w"` or `"a"`)
  does not require `c`.
- Renaming a file requires `d` and `c` on a single granted directory containing
  both the old and new names.
- Temporary files (`io.tmpfile()`, `os.tmpname()`) are created in the system
  temporary directory, so they are only available if that directory is granted
  with `c`.
- In a nested context, each directory must be inside a directory granted by the
  parent context, and its actions are narrowed to those the parent allows.

Restricting file access requires the `iosafe` flag in the context, so functions
that cannot be restricted (e.g. `io.popen`, `os.execute`) are unavailable.  If
the `iosafe` flag is required without granting any directories, no file can be
accessed.

### Working directory

Each context has a working directory, which relative paths resolve against.
The root context's is the process's working directory.  A new context inherits
its parent's unless it sets one, and a relative working directory is relative to
the parent's.

The working directory changes what paths mean, not what is allowed, so:
- it applies to every path golua resolves: `io.open`, `io.lines`, `io.input`,
  `io.output`, `dofile`, `loadfile`, `require`, `os.remove`, `os.rename`,
  `os.tmpname`, `io.tmpfile`;
- `io.popen` and `os.execute` run their command in it;
- it applies even when file access is not restricted, including in `noquotas`
  builds;
- it can be outside every granted directory: relative paths there are simply not
  allowed.

Paths are reported as written, e.g. in error messages and `tostring(file)`,
rather than as resolved, so they don't reveal the host's directory layout.

E.g. to run a script with read access to its document's directory, and that
directory as its working directory, so that `dofile`, `io.open` and `require`
(with the default `package.path`) work relative to the document:
```lua
runtime.callcontext({cwd="stories/cellar", fs={{dir="stories/cellar", allowed="r"}}}, f)
```

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
  -cwd string
    	run as if started in this directory (the script path is relative to it too)
  -flags string
    	compliance flags turned on
  -memlimit uint
    	memory limit
```

E.g. `golua -flags iosafe script.lua` runs `script.lua` with no file access, and
`golua -cwd stories/cellar main.lua` runs `stories/cellar/main.lua` with
`stories/cellar` as its [working directory](#working-directory).

### Within a Lua program

Golua provides a `runtime` library which exposes two functions

#### `runtime.context()`

Returns an object `ctx` representing the current context.  This object mostly
cannot be mutated but gives useful information about the execution context.

- `ctx.status` is the status of the context as a string, which can be:
  - `"live"` if this is the currently running context;
  - `"done"` if this execution context terminated successfully;
  - `"error"` if this execution context terminated with an error
  - `"killed"` if the context terminated because it would otherwise have
    exceeded its limits.
- `ctx.kill` returns an object giving the hard resource limits of `ctx`.  If
  any of these limits are reached then the context will be terminated
  immediately, returning execution to the parent context.  Hard limits cannot
  exceed their parent's hard limits.
- `ctx.stop` returns an object giving the resource soft limits of `ctx`. Soft
  limits cannot exceed hard limits, and by default cannot be increased from the
  parent's soft limits.  In future if there is are clear use-cases for
  increasing the soft limits from the parent's, another API endpoint can be
  provided.
- `ctx.used` returns an object giving the used resources of `ctx`
- `ctx.flags` returns a string describing the flags that any code running in
  this context has to comply with.  Those flags are `"memsafe"`, `"cpusafe"`,
  `"timesafe"` and `"iosafe"` currently.
- `ctx.due` returns true if any of the context's soft limits have been
  exhausted.

Additionally there are two methods that allow mutation of the context.

- `ctx:killnow()` updates the context's state so that its hard limits are
  considered exhausted.  The effect on a running context will be to be
  terminated immediately.
- `ctx:stopnow()` update the context's state so that its soft limits are
  considered exhausted.  The effect is that `ctx.due` returns true.

#### `runtime.callcontext(ctxdef, f, [arg1, ...])`

This function creates a new execution context `ctx` from `ctxdef`, calls
`f(arg1, ...)` in this context, then returns `ctx`. Additionally
- if the call was successful, it also returns the returns values of `f(arg1.
  ...)`;
- if there was a non-terminal error in the call, it also returns the error.  In
  this respect, the `runtime.callcontext()` function always behaves like
  `pcall`.

By default `ctx` will inherit from the current context: its CPU and memory
limits will be the amount of unused CPU and memory in the current context, and
it inherits the `io` and `golib` flags from the current context.

 The argument `ctxdef` allows restricting `ctx` further.  It is a table with any
of the following attributes.
- `kill`: if set, it should be a table.  Attributes can be set in this table
  with names `mem`, `cpu` and values a positive integer.  This is used to set
  the context's hard resource limits.
- `stop`: same format as `kill` but describes soft limits.  It will be used to
  set the context's soft resource limits.
- `flags`: same format as for a context definition (e.g. `"cpusafe memsafe"`)
- `fs`: if set, it should be a list of tables of the form `{dir="path/to/dir",
  allowed="rw"}`, granting access to directories as described in [Restricting
  file access](#restricting-file-access).  An empty list means no file can be
  accessed.  It is an error to grant a directory that the current context cannot
  access.  A relative `dir` is relative to the current context's working
  directory.
- `cwd`: if set, a non-empty string giving the new context's [working
  directory](#working-directory).  A relative `cwd` is relative to the current
  context's working directory.

E.g. to call `f` with read-only access to the `data` directory and full access
to the `out` directory:
```lua
runtime.callcontext({fs={{dir="data", allowed="r"}, {dir="out", allowed="*"}}}, f)
```

Here is a simple example of using this function in the golua repl:
```lua
> ctx = runtime.callcontext({kill={cpu=1000}}, function() while true do end end)
> print(ctx)
killed
> print(ctx.used.cpu, ctx.kill.cpu)
999     1000
> print(ctx.flags)
cpusafe
> print(ctx.used.memory, ctx.kill.memory)
0       nil
```

#### `runtime.killcontext()`

This function terminates the current context immediately, returning to the
parent context.  It is as if a hard resource limit had been hit. It can be used
when a soft resource limit has been hit and the program decides to stop.

Alternatively contexts have a method to achieve the same: `ctx:killnow()`.  On a
context that is not currently running, the effect is to kill it as soon at it is
resumed.
#### `runtime.contextdue()`

This function returns true if any of the soft resource limits has been hit on
the currently running context.

Alternatively contexts have a property `ctx.due` that is set to true if the
context `ctx` has exhausted any of its soft limits.


#### `runtime.stopcontext()`

This function updates the current context so that its soft limits are considered
exhaused.

Alternatively contexts have a method to achieve the same: `ctx:stopnow()`.

### When embedding a runtime in Go

There is a `RuntimeContext` interface in the `runtime` package.  It is
implemented by `*runtime.Runtime` and allows inspection of the current execution
context.  We will see further down that contexts that are terminated are also
available via this interface.

```golang
type RuntimeContext interface {
	HardResourceLimits() RuntimeResources
	SoftResourceLimits() RuntimeResources
	UsedResources() RuntimeResources

	Status() RuntimeContextStatus
	Parent() RuntimeContext

	RequiredFlags() ComplianceFlags

	SetStopLevel(StopLevel)
	Due() bool
}
```

The `runtime` package also defines a `RuntimeContextDef` type whose purpose is
to specify the properties of a new execution context to create.

```golang
type RuntimeContextDef struct {
	HardLimits     RuntimeResources
	SoftLimits     RuntimeResources
	RequiredFlags  ComplianceFlags
	MessageHandler Callable
	FSAccess       *safeio.FSAccess
	WorkDir        string
}
```

`WorkDir` sets the context's [working directory](#working-directory).  If empty,
the parent context's is used; if relative, it is relative to the parent's.

`FSAccess` restricts file access as described in [Restricting file
access](#restricting-file-access).  It must be obtained from the current
context's `FSAccess()` (`safeio.Unrestricted` for a new runtime) and the new
context takes ownership of it, i.e. it is closed when the context is popped.

```golang
access, err := r.FSAccess().Restrict([]safeio.FSGrant{
    {Dir: "data", Actions: safeio.ReadAction},
    {Dir: "out", Actions: safeio.AllActions},
})
if err != nil {
    // e.g. a directory does not exist
}
r.PushContext(rt.RuntimeContextDef{FSAccess: access})
```

`Restrict` resolves relative grant directories against the process's working
directory.  To resolve them against the current context's, use
`safeio.InWorkDir(r, dir)`.

As mentioned above, a Lua runtime is of type `*runtime.Runtime` and implements
the `RuntimeContext` interface.  It also implements two methods.

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
        HardLimits: rt.RuntimeResources{
          Mem: 100000,
          Cpu: 1000000,
        },
        RequiredFlags: rt.ComplyIoSafe
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

#### `(*Runtime).CallContext(def RuntimeContextDef, f func() *Error) (RuntimeContext, *Error)`

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

    ctx, err := r.CallContext(rt.RuntimeContextDef{
        HardLimits: rt.RuntimeResources{
          Mem: 100000,
          Cpu: 1000000,
        },
        RequiredFlags: rt.ComplyIoSafe
    }, func() *rt.Error {
        // Do something in this context, returning an error if appropriate.
        // That error will set the context status to "error".
    })

    // Panics due to quota exceeded will be recovered from.
}
```

#### `(*Runtime).TerminateContext(format string, args ...interface{})`

Terminate the context immediately if it is live.

## Finalizers and runtime contexts

In Lua it is possible to add finalizers to two types of values: tables and
userdata.  Finalizers are run once the garbage collector knows the values are no
longer reachable.  This is used in the standard library to make sure open files
which are no longer used are closed.

The Golua runtime makes sure that when a value is created within a runtime
context with restricted resources, running its finalizer will not use another
context's resources.  However, only in the case of userdata,  it is sometimes
the case the value contains a resource that should be released unconditionnally
(e.g. a file descriptor).  Golua provides a general mechanism to support that,
simply by defining a `ReleaseResources` method on the underlying type.  That
method is guaranteed to run before the runtime context is closed, but after the
Lua finalizer runs if it exists.  For example in the standard library, the
underlying type of file userdata is as follows.

```golang
type File struct {
	file   *os.File
	status fileStatus
	reader bufReader
	writer bufWriter
}
```

When file userdata becomes unreachable, the underlying `File` instance should
close the `os.File` instance it owns.  This is done as follows.

```golang

// The *File type implements the ResourceReleaser interface.
func (f *File) ReleaseResources(d *rt.UserData) {
	f.cleanup()
}

// Included to show what happens in Prefinalize
func (f *File) cleanup() {
	if !f.IsClosed() {
		f.Close()
	}
	if f.IsTemp() {
		_ = os.Remove(f.Name())
	}
}
```

The runtime makes sure that any userdata that implements
`runtime.ResourceReleaser` interface will have its `ReleaseResources` method
called unconditionally.  Of course it is important that the code in those
methods consumes as little resources as possible.

For details about the semantics see the
[userdata.quotas.lua](runtime/lua/userdata.quotas.lua) test file

## How to implement the safe execution environment

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

### Restricting access to Go functions.

There is a built-in mechanism for making sure that Go function called in the Lua
runtime comply with the safe execution environment requirements.  As there are
different levels of compliance, a number of Compliance Flags can be defined.
Any of those can be required in an execution context, and only Go functions
which have been declared explicitly as implementing these compliance flags will
be allowed to be run.

This approach has several advantages
- Granularity: for each Go function it is required to define what compliance
  flags it implements.  So a single Lua module could include Go functions with
  different compliance profiles.
- Future proof: if new compliance flags are added, existing functions will not
  comply with those by default, so it limits the risk of misuse.  On the other
  hand an existing function will still be able to be used in an environment not
  requiring the new compliance flags.
- Safety: It is safer than controlling access to modules via a
  blocklist/allowlist.  As Lua's runtime is very dynamic, it would probably be
  easy to circumvent such measures.

#### `ComplianceFlags`

The runtime defines a number of compliance flags, currently:

```golang

type ComplianceFlags uint16

const (
	// Only execute code checks memory availability before allocating memory
	ComplyMemSafe ComplianceFlags = 1 << iota

	// Only execute code that checks cpu availability before executing a
	// computation.
	ComplyCpuSafe

	// Only execute code that complies with IO restrictions, i.e. functions
	// that do no IO or only access the filesystem via the safeio package.
	ComplyIoSafe
)
```

A Go function declaring `ComplyIoSafe` must access files only via the functions
in the `safeio` package (`safeio.OpenFile`, `safeio.RemoveFile`,
`safeio.RenameFile`, `safeio.TempFile`), passing them the `*rt.Runtime`, so that
the current context's file access restrictions are enforced, and relative paths
are resolved against its working directory.

#### `(*GoFunction).SolemnlyDeclareCompliance(ComplianceFlags)`

Any Go functions that can be called from Lua is wrapped in an instance of
`*rt.GoFunction`.  By default this instances does not include any compliance
flags.  It is possible to declare compliance with
`(*GoFunction).SolemnlyDeclareCompliance()`

Before execution, the current context's `RequiredFlags` value is checked against
the compliance flags declared by the Go functions.  If any of the required flags
is not complied with by the function, execution will immediately return an error
(but not terminate the context).
