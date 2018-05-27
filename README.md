# GoLua

Implementation of Lua in Go.

# Quick start

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

## Known unsolved issues

* `collectgarbage()`. Probably a noop?

## Roadmap

### Lexer / Parser

Done. Note a custom lexer is required to tokenise long strings and
long comments.

* The lexer is implemented in the package `scanner`.
* The parser is generated from `lua.bnf` using gocc
  (https://github.com/goccmack/gocc). The command used is:
  `gocc -no_lexer lua.bnf`.

### AST -> IR Compilation

Done

* Each node in the AST (package `ast`) knows how to compile itself
  using an `ir.Compiler` instance.
* IR instructions and compiler are defined in the `ir` package.

### IR -> Code Compilation

Done

* Each IR instruction (package `ir`) know how to compiler itself using
  an `ir.ConstantCompiler` instance (TODO: rename this type).
* Runtime bytecode is defined in the `code` package

### Runtime

Mostly done.  To do
* tables: deleting entries with nil values
* implementing weak tables (can it even be done?)
* nil: decide between NilType{} and nil (currently both work)

* The runtime is implemented in the `runtime` package.

### Test Suite

Framework done. In the directory `luatest/lua`, each `.lua` file is a
test. Expected output is specified in the file as comments of a
special form, starting with `-->`:

```lua
print(1 + 2)
--> =3
-- "=" means match literally the output line

print("ababab")
--> ~^(ab)*$
-- "~" means match with a regexp (syntax is go regexp)
```

TODO: write a lot more tests

### Standard Library

* basic library: done apart from `xpcall`
* coroutine library: done
* package library: loading lua modules done - think about loading go
  modules, perhaps using the plugin mechanism
  (https://golang.org/pkg/plugin/)
* string library: `byte`, `char`, `len`, `lower`, `upper`, `reverse`, `sub` done
* utf8 library: TODO
* table library: `concat`, `insert`, `move` done
* math library: TODO
* io library: TODO
* os library: TODO
* debug library (I don't know how much of this can reasonably be
  implemented as I didn't want to be constrained by it when designing
  golua)
