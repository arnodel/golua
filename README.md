GoLua
=====

Implementation of Lua in Go. Roadmap below.

Lexer / Parser
--------------

Done

AST -> IR Compilation
---------------------

Almost there.  To do:
* break
* label / goto
* upvalue mutations

IR -> Code Compilation
----------------------

To do

Runtime
-------

Started (Thread and Continuation).  To do next:
* LuaContinuation - implementation of Continuation running from
  compiled Lua bytecode.

Standard Library
----------------

To do
