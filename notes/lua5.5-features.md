# Lua 5.5 Features Implementation Checklist

Based on [Lua 5.5 README](https://www.lua.org/manual/5.5/readme.html) and [Incompatibilities](https://www.lua.org/manual/5.5/manual.html#8).

## Language Features

- [x] **Global variable declarations** - New syntax for declaring global variables
  - Syntax: `global [attrib] name [attrib]` or `global [attrib] *` (wildcard)
  - Attributes: `<const>` makes globals read-only
  - Semantics:
    - Outside any global declaration: Lua works as "global-by-default" (legacy Lua 5.4 mode)
    - Inside any global declaration: No default - all globals must be explicitly declared
    - Wildcard `global *` declares all undeclared names as mutable globals
    - Wildcard `global<const> *` declares all undeclared names as const globals
  - Implementation phases:
    1. Lexing: Add `global` keyword token
    2. Parsing: Parse global statements into AST
    3. Compilation: Track declarations per scope and validate access (reads/writes)
  - Complexity: **Medium** - requires changes across lexer, parser, compiler

- [ ] **Read-only for-loop variables** - Loop control variables cannot be modified within the loop body
  - Affects: `for i = 1, 10 do` and `for k, v in pairs(t) do`
  - Implementation: Add compile-time check to prevent assignment to loop control variables
  - Complexity: **Low** - similar to const local variable checking

- [ ] **Named vararg tables** - Enhanced variable argument handling
  - Syntax: `function f(...name)` where `name` is optional
  - Semantics:
    - The name refers to a read-only local variable that refers to the vararg table
    - Optimization: If the vararg table isn't captured as an upvalue, no actual table is created
    - Indexing expressions and vararg expressions are translated to direct vararg data access
  - Example: `function f(...args) return args[1] end`
  - Implementation:
    - Parser: Accept optional name after `...`
    - Compiler: Create read-only local variable for named vararg
    - Runtime: Optimize when vararg table isn't used as upvalue
  - Complexity: **Medium-High** - affects function parameter handling and optimization

## Standard Library Functions

- [ ] **`table.create(nseq [, nrec])`** - New function for creating preallocated tables
  - Signature: `table.create(nseq [, nrec])`
  - Parameters:
    - `nseq`: Hint for how many sequence elements (array part) the table will have
    - `nrec`: (optional) Hint for how many hash elements the table will have (defaults to 0)
  - Behavior: Creates empty table with preallocated memory based on hints
  - Purpose: Performance optimization to avoid repeated reallocations
  - Implementation: Add to `lib/tablelib`
  - Complexity: **Low** - straightforward library function addition

- [ ] **Enhanced `utf8.offset`** - Returns both start and end positions
  - Old behavior: Returned single position (start of nth character)
  - New behavior: Returns two integers - start position and end position of character encoding
  - Return value: `(start_pos, end_pos)` in bytes
  - Special case: If character is right after end of string, behaves as if there's a '\0' there
  - Implementation: Modify existing function in `lib/utf8lib` to return two values
  - Complexity: **Low** - simple function modification

- [N/A] **Garbage collection parameter changes** - New "param" option system
  - Not applicable: Go runtime manages GC, not under our control
  - The `collectgarbage()` function exists but defers to Go's GC
  - No action needed

## Runtime Improvements

- [x] **Float printing improvements** - Floats printed in decimal with enough digits to read back correctly
  - Status: Already implemented correctly in Go (tested: `0.1 + 0.2` prints `0.30000000000000004`)
  - No action needed

- [N/A] **Incremental major garbage collection** - Major GC cycles done incrementally
  - Not applicable: Go runtime manages GC, not under our control

- [x] **Memory-efficient arrays** - Large arrays use ~60% less memory
  - Status: Already implemented - `mixedTable` has separate `array` and `hashTable` parts
  - See: `runtime/hashtable.go:21` - `type mixedTable struct`
  - No action needed

- [N/A] **Constructor depth expansion** - Support for more levels in table constructors
  - Not applicable: Go implementation doesn't enforce limits on table constructor nesting
  - No action needed

- [N/A] **External strings** - Strings using memory not managed by Lua
  - Not applicable: Go manages string memory, not under our control

- [ ] **`__call` metamethod chain limit** - Maximum 15 objects in chain
  - Implementation: Add counter in function call evaluation during metamethod resolution
  - Prevents infinite recursion through chained `__call` metamethods
  - Complexity: **Low-Medium** - add depth tracking to call handling

- [ ] **Nil error objects replaced with message** - Error handling behavior change
  - Implementation: When nil becomes an error object, replace it with a string message
  - Affects: Error propagation and handling code
  - Complexity: **Low-Medium** - modify error handling logic

## Go API

- [x] **Selective library loading** - Equivalent to `luaL_openselectedlibs`
  - Status: Already implemented via `lib.LoadLibs(r, ...loaders)`
  - See: `lib/lib.go:19` - accepts variadic loader arguments
  - Users can call `LoadLibs` with only the libraries they want
  - No action needed
