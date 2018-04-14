GoLua
=====

Implementation of Lua in Go. Roadmap below.

Lexer / Parser
--------------

Done

AST -> IR Compilation
---------------------

Almost there.  To do:
* label / goto
* upvalue mutations - is that a runtime thing though?

IR -> Code Compilation
----------------------

Done, AFAICS.

Runtime
-------

Started (Thread and Continuation).  To do next:
* LuaContinuation - implementation of Continuation running from
  compiled Lua bytecode.

Standard Library
----------------

To do
