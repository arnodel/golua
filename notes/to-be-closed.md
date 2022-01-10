# Implementation notes for to-be-closed variables

This is a feature introduced in Lua 5.4.  See
https://www.lua.org/manual/5.4/manual.html#3.3.8

## Approach

I try to give a succinct explanation of the implementation approach.

### Height of a lexical scope

During AST -> IR compilation, a **height** is assigned to each lexical scope.
If L2 is a lexical scope with parent L1, then

    height(L2) = height(L1) + n, where n is the number of to-be-closed variables that L2 defines

If L is a root lexical scope (i.e. it is the outermost block in a function body)
then

    height(L) = 0

### Code generation

There are 2 new opcodes:
- `clpush <reg>`: push the contents of `<reg>` onto the close stack
- `cltrunc h`: truncate the close stack to height `h` (`h` is encoded in the
  opcode and must be known at compile time)

This is how the opcodes are inserted
- Whenever a to-be-closed varable is defined (`local x <close> = val`), a
  `clpush r1` instruction is emitted, where `r1` is the register containing
  `val`.
- Whenever a lexical scope L is exited, a `cltrunc h` instruction is emitted where `h` is the height of the parent scope of `L`
- Whenever a Jump is emitted, just before the jump is emitted a `cltrunc h`
  instruction is emitted where `h` is the height of the lexical scope of the
  jump destination.

### Runtime

There are two aspects to the runtime machinery - normal execution of Lua
continutaions and error handling

#### Normal execution of Lua continuations

Each Lua continuation maintains a **close stack**, which is a stack of Lua
values. It starts empty and is modified as follows
- `clpush <reg>` pushes the value of `<reg>` on top of the close stack
- `cltrunc h` pops values from the top of the close stack and executes their
  `__close` metamethod until the height of the close stack is at most `h`.
- When returning from a Lua function (return OR tail-call), a `cltrunc 0`
  instruction is executed.

#### Error handling

If there is an error, then all close stacks in the "call stack" should be called
until the error is caught.  This is achieve by adding a `Cleanup()` method to
the `runtime.Cont` interface, which must be implemented by each continuation
type.   The runtime loop will call those in turn down the call stack when an
error is encountered.  For a Lua continuation, `Cleanup()` is roughly equivalent
to `cltrunc 0`.

#### Coroutines

Lua 5.4 adds a `coroutine.close()` function that allows stopping a suspended
coroutine but executing all pending to-be-closed variables.  Given the approach
above this is simply a matter of executing the `Cleanup()` method on the
suspended thread's current continuation and its successors.

