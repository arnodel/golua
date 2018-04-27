GoLua
=====

Implementation of Lua in Go. Roadmap below.

Hello world is now running!!!

Lexer / Parser
--------------

This almost works apart from:
* long strings (e.g. `[===[ ... ]===]`)
* long comments (e.g. `-- [=[ ... ]=]`)

They would require writing a custom lexer rather than generating one
with gocc though (good resource:
https://talks.golang.org/2011/lex.slide#1)

AST -> IR Compilation
---------------------

Almost there.  To do:
* upvalue mutations - is that a runtime thing though?

IR -> Code Compilation
----------------------

Done, AFAICS.

Runtime
-------

Mostly done.  To do
* testing
* implementing cells / mutable upvalues

Test Suite
----------

Now done. In the directory `luatest/lua`, each `.lua` file is a
test. Expected output is specified in the file as comments of a
special form, starting with `-->`:

```lua
print(1 + 2)
--> =3
-- "=" means match literally the output line

print("abaabab")
--> ~^(ab)*$
-- "~" means match with a regexp (syntax is go regexp)
```


Standard Library
----------------

The basic library is partly done.

The coroutine library has `create`, `resume` and `yield` implemented
and tested so far.
