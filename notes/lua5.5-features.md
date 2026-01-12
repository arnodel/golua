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

- [x] **Named vararg tables** - Enhanced variable argument handling with shared mutation semantics
  - Syntax: `function f(...name)` where `name` is optional
  - Semantics:
    - The name refers to a read-only local variable that refers to the vararg table
    - **Shared mutation**: Modifying `args[i]` within original bounds affects what `...` expands to
    - **Array growth divergence**: Adding beyond original bounds creates a copy (mutations diverge)
    - Upvalue capture: Table survives function lifetime when captured (Go GC handles this)
  - Example: `function f(...args) args[1] = 999; print(...) end  -- prints 999, ...`
  - Status: **Implemented** with proper Lua 5.5 semantics
  - Implementation approach:
    - New opcode `OpMkVarargTable` (Type4a) creates table from vararg data
    - `NewTableFromSlice()` constructor creates table whose array part references vararg slice (no copy)
    - Go's GC keeps slice alive as long as table exists (automatic upvalue safety)
  - Files modified:
    - `code/opcodes.go` - Added `OpMkVarargTable` opcode and disassemble case
    - `code/instructions.go` - Added `MkVarargTable()` instruction builder
    - `ir/instructions.go` - Added `MkVarargTable` IR instruction type
    - `ircomp/compinstr.go` - IR to bytecode compilation
    - `runtime/table.go` - Added `NewTableFromSlice()` constructor
    - `runtime/luacont.go` - Runtime execution of new opcode
    - `astcomp/compexp.go` - Use `MkVarargTable` instead of `MkTable + FillTable`
    - `runtime/lua/named_varargs.lua` - 12 comprehensive tests including shared mutation
  - Complexity: **Medium** - required new opcode but leveraged existing infrastructure

## Standard Library Functions

- [x] **`table.create(nseq [, nrec])`** - New function for creating preallocated tables
  - Signature: `table.create(nseq [, nrec])`
  - Parameters:
    - `nseq`: Hint for how many sequence elements (array part) the table will have
    - `nrec`: (optional) Hint for how many hash elements the table will have (defaults to 0)
  - Behavior: Creates empty table with preallocated memory based on hints
  - Purpose: Performance optimization to avoid repeated reallocations
  - Implementation: Added to `lib/tablelib` with conditional preallocation
    - When memory quotas active (hard or soft): No preallocation (security safe)
    - When no memory quotas: Preallocates capacity (performance optimization)
    - See [table-memory-accounting.md](table-memory-accounting.md) for details on discovered issue
  - Complexity: **Low** - straightforward library function addition
  - Files modified:
    - `runtime/hashtable.go` - Added `newMixedTableWithCapacity(nseq, nrec int)`
    - `runtime/table.go` - Added `NewTableWithCapacity(nseq, nrec int)`
    - `lib/tablelib/tablelib.go` - Implemented `create` function
    - `lib/tablelib/lua/tablelib.lua` - Added comprehensive tests
    - `lib/tablelib/lua/tablelib.quotas.lua` - Added quota tests
  - **Note**: Once comprehensive table memory accounting is implemented, revisit to enable preallocation with quotas

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
