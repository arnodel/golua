# GoLua

Implementation of Lua in Go.

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
* mutable upvalues.  Easy if willing to sacrifice performance.
* long string / comments.  Requires writing a custom lexer (see
  below).

## Roadmap

### Lexer / Parser

This almost works apart from:
* long strings (e.g. `[===[ ... ]===]`)
* long comments (e.g. `-- [=[ ... ]=]`)

They would require writing a custom lexer rather than generating one
with gocc though (good resource:
https://talks.golang.org/2011/lex.slide#1)

### AST -> IR Compilation

Almost there.  To do:
* upvalue mutations - is that a runtime thing though?

### IR -> Code Compilation

Done

### Runtime

Mostly done.  To do
* implementing cells / mutable upvalues
* tables: deleting entries with nil values
* implementing weak tables (can it even be done?)

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

The basic library is done apart from `xpcall`.

The coroutine library has `create`, `resume` and `yield` implemented
and tested so far.

TODO:
* package library
* string library
* utf8 library
* table library
* math library
* io library
* os library
* debug library (I don't know how much of this can reasonably be
  implemented as I didn't want to be constrained by it when designing
  golua)
