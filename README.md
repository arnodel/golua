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

Standard Library
----------------

To do
