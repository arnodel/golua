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
* long string / comments.  Requires writing a custom lexer (see
  below).

## Roadmap

### Lexer / Parser

Done. Note a custom lexer is required to tokenise long strings and
long comments.

* The lexer is implemented in the package `scanner`.
* The parser is generated from `lua.bnf` using gocc
  (https://github.com/goccmack/gocc). The command used is `gocc -a
  lua.bnf`.

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
* package library: TODO
* string library: TODO
* utf8 library: TODO
* table library: TODO
* math library: TODO
* io library: TODO
* os library: TODO
* debug library (I don't know how much of this can reasonably be
  implemented as I didn't want to be constrained by it when designing
  golua)
