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
- Whenever a lexical scope L is exited, a `cltrunc h` instruction is emitted
  where `h` is the height of the parent scope of `L`
- Whenever a Jump is emitted, just before the jump is emitted a `cltrunc h`
  instruction is emitted where `h` is the height of the lexical scope of the
  jump destination.

### Runtime

There are two aspects to the runtime machinery - normal execution of Lua
continutaions and error handling

#### Normal execution of Lua continuations

Each Lua thread maintains a **close stack**, which is a stack of Lua
values. It starts empty and is modified as follows
- when a Lua continuation is created it records the current height of the close
  stack as an offset, called `base`.
- `clpush <reg>` pushes the value of `<reg>` on top of the close stack.
- `cltrunc h` pops values from the top of the close stack and executes their
  `__close` metamethod until the height of the close stack is at most `h + base`.
- When returning from a Lua function, a `cltrunc 0` instruction is executed.
- tail-calls are forbidden if there are to-be-closed values pending at the point
  of the tail-call, because they need to be run after returning from the call.

#### Error handling

If there is an error, then all close stacks in the "call stack" should be called
until the error is caught.  The `Thread.CallContext` function makes sure the
pending to-be-closed values are called after execution has finished, whether
there was an error or not.

#### Coroutines

Lua 5.4 adds a `coroutine.close()` function that allows stopping a suspended
coroutine but executing all pending to-be-closed variables.  Given the approach
above this is simply a matter of executing the `cleanupCloseStack()` method on
the suspended thread.


#### Continuations

A continuation (the `cont` library) can be called after the blocks it was
captured in have ended, or while other blocks are running, so the values that
are pending no longer form one stack: they form a tree, of which the thread's
close stack is the branch that is running.

- The close stack is a linked list from its top down, and its height counts
  every value pushed.  Only the values that are to be closed in fact (not
  `nil` or `false`, which is what almost every `for` loop pushes) are in the
  list, each with the height of the stack it is the top of.
- A continuation keeps the close stack as it is when it is captured: a top and
  a height.  Calling it makes that the thread's close stack.
- The two stacks have in common the values that were pending before they
  parted.  The blocks of the others are left by the call.  A block that is left
  is over if no continuation was captured in it, as nothing can run it again,
  and then its value is closed, as `break` or `return` would close it.  If one
  was, the block can be entered again, by calling that continuation: its value
  stays pending and is closed when the block ends, as that of a coroutine
  which has yielded.  So capturing a continuation marks the values that are
  pending.
- Several copies of a frame can end the same block, each when it gets there.
  The value is closed once.
- An error closes every pending value down to where it is caught, as before,
  and so does a continuation that leaves Go code which was waiting for Lua
  code (a metamethod call, a function given to `table.sort`): nothing under
  that Go code can be entered again.  A `pcall` is not such Go code.  The
  function it calls can be entered again after a continuation has left it,
  so it is a block like any other here.
- A coroutine without a goroutine is given one before it declares a value
  to be closed in fact (see `runtime/lightthread.go`), so calling a
  continuation there never has a `__close` handler to call that the
  coroutine could not run.

This is the "wind list" of Scheme's `dynamic-wind`, of which a value to be
closed uses half: there is something to do when a block is left for good, and
nothing to do when it is entered again.
