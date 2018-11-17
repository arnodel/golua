[![Build Status](https://travis-ci.com/arnodel/golua.svg?branch=master)](https://travis-ci.com/arnodel/golua)
[![Go Report Card](https://goreportcard.com/badge/github.com/arnodel/golua)](https://goreportcard.com/report/github.com/arnodel/golua)
[![Coverage](https://codecov.io/gh/arnodel/golua/branch/master/graph/badge.svg)](https://codecov.io/gh/arnodel/golua)

# GoLua

Implementation of Lua **5.3** in Go.

## Quick start: running golua

To install, run:

```sh
$ go get github.com/arnodel/golua
```

To run interactively (in a repl):

```
$ golua
> function fac(n)
|   if n == 0 then
|     return 1
|   else
|     return n * fac(n - 1)
|   end
| end
> -- For convenience the repl also evaluates expressions
> -- and prints their value
> fac(10)
3628800
> for i = 0, 5 do
|   print(i, fac(i))
| end
0	1
1	1
2	2
3	6
4	24
5	120
>
```

To run a lua file:

```sh
$ golua myfile.lua
```

Or

```sh
cat myfile.lua | golua
```

E.g. if the file `myfile.lua` contains:

```lua
local function counter(start, step)
    return function()
        local val = start
        start = start + step
        return val
    end
end

local nxt = counter(5, 3)
print(nxt(), nxt(), nxt(), nxt())
```

Then:

```sh
$ golua myfile.lua
5	8	11	14
```

Errors produce useful tracebacks, e.g. if the file `err.lua` contains:

```lua
function foo(x)
    print(x)
    error("do not do this")
end

function bar(x)
    print(x)
    foo(x*x)
end

bar(2)
```

Then:

```
$ golua err.lua
2
4
!!! error: do not do this
in function foo (file err.lua:3)
in function bar (file err.lua:8)
in function <main chunk> (file err.lua:11)
```

## Quick start: embedding golua

It's very easy to embed the golua compiler / runtime in a Go program.  The example below compiles a lua function, runs it and displays the result.

```golang
	// First we obtain a new Lua runtime which outputs to stdout
	r := rt.New(os.Stdout)

	// This is the chunk we want to run.  It returns an adding function.
	source := []byte(`return function(x, y) return x + y end`)

	// Compile the chunk. Note that compiling doesn't require a runtime.
	chunk, _ := rt.CompileAndLoadLuaChunk("test", source, r.GlobalEnv())

	// Run the chunk in the runtime's main thread.  Its output is the Lua adding
	// function.
	f, _ := rt.Call1(r.MainThread(), chunk)

	// Now, run the Lua function in the main thread.
	sum, _ := rt.Call1(r.MainThread(), f, rt.Int(40), rt.Int(2))

	// --> 42
	fmt.Println(sum)
```

## Quick start: extending golua

It's also very easy to add write Go functions that can be called from Lua code.
The example below shows how to.

This is the Go function that we are going to call from Lua. Its inputs are:
- `t`: the thread the function is running in.
- `c`: the go continuation that represents the context the function is called in.
       It contains the arguments to the function and the next continuation (the
       one which receives the values computed by this function).

 It returns the next continuation on success, else an error.

```golang
func addints(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var x, y rt.Int

	// First check there are two arguments
	err := c.CheckNArgs(2)
	if err == nil {
		// Ok then try to convert the first argument to a lua integer (rt.Int).
		x, err = c.IntArg(0)
	}
	if err == nil {
		// Ok then try to convert the first argument to a lua integer (rt.Int).
		y, err = c.IntArg(1)
	}
	if err != nil {
		// Some error occured, we return it in our context
		return nil, err.AddContext(c)
	}
	// Arguments parsed!  First get the next continuation.
	next := c.Next()

	// Then compute the result and push it to the continuation.
	next.Push(x + y)

	// Finally return the next continuation.
	return next, nil

	// Note: the last 3 steps could have been written as:
	// return c.PushingNext(x + y), nil
}
```

The code sample below shows how this function can be added to the Lua runtime
environment and demonstrates calling it from Lua.

```golang
	// First we obtain a new Lua runtime which outputs to stdout
	r := rt.New(os.Stdout)

	// Load the basic library into the runtime (we need print)
	base.Load(r)

	// Then we add our addints function to the global environment of the
	// runtime.
	rt.SetEnvGoFunc(r.GlobalEnv(), "addints", addints, 2, false)

	// This is the chunk we want to run.  It calls the addints function.
	source := []byte(`print("hello", addints(40, 2))`)

	// Compile the chunk.
	chunk, _ := rt.CompileAndLoadLuaChunk("test", source, r.GlobalEnv())

	// Run the chunk in the runtime's main thread.  It should output 42!
	_, _ = rt.Call1(r.MainThread(), chunk)
```

## Aim

To implememt the Lua programming language in Go, easily embeddable in
Go applications.  It should be able to run any pure Lua code

## Design constraints

* clean room implementation: do not look at existing implementations
* self contained: no dependencies
* small: avoid re-implementing features which are already present in
  the Go language or in the standard library (e.g. garbage collection)
* register based VM
* no call stack (continuation passing)

## Components

### Lexer / Parser

* The lexer is implemented in the package `scanner`.
* The parser is hand-written and implemented in the `parsing` package.

### AST → IR Compilation

The `ast` package defines all the AST nodes. Each node in the AST
knows how to compile itself using an `ir.Compiler` instance.

The `ir` package defines all the IR instructions and the IR compiler.

### IR → Code Compilation

The runtime bytecode is defined in the `code` package.  Each IR
instruction (see package `ir`) know how to compile itself using an
`ir.InstrCompiler` instance.

### Runtime

The runtime is implemented in the `runtime` package.  This defines a
`Runtime` type which contains the global state of a runtime, a
`Thread` type which can run a continuation, can yield and can be
resumed, the various runtime data types (e.g. `String`, `Int`...). The
bytecode interpreter is implemented in the `RunInThread` method of the
`LuaCont` data type.

### Test Suite

There is a framework for running lua tests in the package `luatests`.
In the directory `luatests/lua`, each `.lua` file is a test. Expected
output is specified in the file as comments of a special form,
starting with `-->`:

```lua
print(1 + 2)
--> =3
-- "=" means match literally the output line

print("ababab")
--> ~^(ab)*$
-- "~" means match with a regexp (syntax is go regexp)
```

There is good coverage of the standard library but more tests need to
be written for the core language.

### Standard Library

The `lib` directory contains a number of package, each implementing a
lua library.

* `base`: basic library. It is done apart from `xpcall` (and the
  implementation of `load` is not complete).
* `coroutine`: the coroutine library, which is done.
* `packagelib`: the package library.  It is able to load lua modules
  but not "native" modules, which would be written in Go. Obviously
  this is not part of the official Lua specification. Perhaps using
  the plugin mechanism (https://golang.org/pkg/plugin/) would be a way
  of doing it.  I have no plan to support Lua C modules!
* `stringlib`: the string library.  It is complete.
* `mathlib`: the math library,  It is complete.
* `tablelib`: the table library.  It is complete.
* `iolib`: the io library.  It is implemented apart from `popen`,
  `file:setvbuf`, `read("n")` (reading a number)
* `utf8lib`: the utf8 library.  It is complete.

The following libraries do not exist at all:
* os library
* debug library (I don't know how much of this can reasonably be
  implemented as I didn't want to be constrained by it when designing
  golua)
