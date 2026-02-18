# Golua 5.5 Conformance Notes

This document lists known behavioral differences between Golua and reference Lua 5.5, discovered while adapting the official Lua test suite. Each issue has a unique code (e.g., `GOLUA-001`) that is referenced in the [golua-tests](https://github.com/arnodel/golua-tests/tree/golua-5.5) source code where tests are skipped or adapted.

---

## Bugs (To Fix)

These are correctness issues that should be fixed.

| Code | Category | Summary |
|------|----------|---------|
| GOLUA-017 | Parser | Duplicate labels rejected in different scopes |

---

---

## Accepted Differences

These are minor incompatibilities or intentional design choices that we accept.

| Code | Category | Summary |
|------|----------|---------|
| GOLUA-008 | Semantic | Jumping over `global *` allowed |
| GOLUA-009 | Runtime | Unary metamethods receive 1 arg (not 2) |
| GOLUA-011 | Runtime | Coroutine close behavior differs |
| GOLUA-012 | Runtime | Stack overflow detection differs |
| GOLUA-013 | Runtime | Yielding allowed in more contexts |
| GOLUA-014 | Runtime | `debug.getinfo` differences |
| GOLUA-015 | Runtime | Weak table GC timing differs |
| GOLUA-016 | Runtime | `debug.upvalueid` fails for Go functions |
| GOLUA-020 | Runtime | Random number generator differs |
| GOLUA-021 | Runtime | `math.random` accepts extra arguments |
| GOLUA-031 | Runtime | GC memory counting differs |
| GOLUA-033 | Runtime | Infinite coroutine creation not detected |
| GOLUA-037 | Runtime | `table.move` iteration order differs |
| GOLUA-039 | Runtime | `table.sort` doesn't detect invalid order functions |
| GOLUA-040 | Runtime | File metatable `__name` is "file" not "FILE*" |
| GOLUA-041 | Runtime | `load()` accepts "B" mode (no fixed buffer concept) |
| GOLUA-043 | Runtime | `io.lines` accepts unlimited arguments |

---

## Error Message Differences

These are cosmetic differences in error message wording. Tests have been adapted to accept both forms.

| Code | Golua Message | Lua Message |
|------|---------------|-------------|
| GOLUA-024 | "#1 must be a number" | "number expected" |
| GOLUA-025 | "value needed" | "value expected" |
| GOLUA-026 | "attempt to perform 'n%0'" | "...zero..." |
| GOLUA-027 | "number has no integer representation" | "number (field 'huge') has no integer representation" |
| GOLUA-028 | "assign to const global variable" | "assign to const variable" |
| GOLUA-029 | "global: only <const> is allowed for attribute" | "global variable cannot be to-be-closed" |
| GOLUA-030 | "reassign constant variable" | "assign to const variable" |
| GOLUA-032 | "cannot close a running thread" | "cannot close a main coroutine" |
| GOLUA-034 | "too many values to unpack" | "too many results to unpack" |
| GOLUA-035 | "len should return an integer" | "object length is not an integer" |
| GOLUA-036 | "#1 must be a table" | "table expected" |
| GOLUA-038 | "interval too large" | "too many elements to move" |
| GOLUA-042 | "file already closed" | "file is already closed" |
| GOLUA-044 | "illegal character" | "a binary chunk" |
| GOLUA-045 | "unknown directive" | "invalid conversion specifier" |

---

## Files Disabled Due to Extensive Differences

- **errors.lua**: Error messages differ significantly throughout
- **db.lua**: Debug library has differences
- **gengc.lua**: Go uses its own GC, no generational GC
- **cstack.lua**: Stack overflow detection works differently

---

## Fixed Issues

| Code | Summary |
|------|---------|
| ~~GOLUA-001~~ | `local<const>` prefix syntax - was stale binary, parser supports it |
| ~~GOLUA-003~~ | `global X` doesn't shadow `local X` - fixed in `ir/context.go` |
| ~~GOLUA-005~~ | Strict mode doesn't propagate to nested functions - fixed in `ir/builder.go` |
| ~~GOLUA-006~~ | `global function` doesn't shadow local - fixed with GOLUA-003 |
| ~~GOLUA-018~~ | Float-to-integer conversion too permissive - fixed in `runtime/numconv.go` |
| ~~GOLUA-019~~ | `math.frexp`/`math.ldexp` not implemented - fixed in `lib/mathlib` |
| ~~GOLUA-022~~ | `tostring` doesn't preserve `.0` suffix - fixed in `runtime/value.go` and `lib/stringlib/format.go` |
| ~~GOLUA-004~~ | `_ENV` can be declared as global - fixed in `astcomp/compstat.go` |
| ~~GOLUA-023~~ | `os.execute` not implemented - fixed in `lib/oslib/oslib.go` |
| ~~GOLUA-007~~ | Global redefinition not prevented - fixed with `CheckNotDefined` instruction |
| ~~GOLUA-010~~ | Vararg table unpacking ignores `n` field - fixed in `runtime/luacont.go` |
| ~~GOLUA-002~~ | `global` keyword always reserved - fixed with context-sensitive parsing in `parsing/parser.go` |
| Hashtable panic | Fixed in `runtime/hashtable.go` - capacity now rounded to power of 2 |

---
---

# Detailed Descriptions

## Bugs (To Fix)

### GOLUA-017: Duplicate labels rejected even in different scopes
**Severity**: Stricter than Lua
**Test file**: goto.lua

Golua rejects labels with the same name even when they're in different scopes. Lua allows this.

```lua
local function testG(a)
  if a == 1 then
    goto l1
  elseif a == 4 then
    goto l1     -- Different scope, same name
    ::l1:: a = a + 1  -- Error in Golua: duplicate label
  end
  ::l1:: return "1"   -- Original l1
end
```

---

### GOLUA-018: Float-to-integer conversion too permissive
**Severity**: Bug
**Test file**: math.lua

Golua's float-to-integer conversion for bitwise operations doesn't properly detect numbers that can't be exactly represented as integers.

```lua
-- 2.0^63 is larger than maxint but golua converts it
local result = 2.0^63 & 1  -- Should error, golua returns 1

-- maxint + 0.0 loses precision as float, should fail conversion
local x = math.maxinteger + 0.0
local y = x | x  -- Should error, golua returns maxint
```

Also affects `math.tointeger`:
```lua
math.tointeger(0.0 - math.mininteger)  -- Should return nil, golua returns maxint
```

---

## Missing Features

### GOLUA-023: `os.execute` not implemented
**Test file**: main.lua

```lua
print(os.execute)  -- nil
```

---

## Intentional / Low Priority Differences

### GOLUA-009: Unary metamethods receive 1 argument (not 2)
**Test file**: events.lua

Lua passes 2 identical arguments to unary metamethods (`__unm`, `__len`, `__bnot`) as an implementation detail to simplify its VM internals. The Lua manual states this "may be removed in future versions."

Golua passes only 1 argument, which is the semantically correct number for a unary operation.

```lua
local mt = {
  __unm = function(...)
    print(select('#', ...))  -- Lua: 2, Golua: 1
  end
}
local a = setmetatable({}, mt)
local b = -a
```

---

### GOLUA-008: Jumping over `global *` is allowed
**Test file**: goto.lua:34-36

Lua 5.5 prevents goto from jumping over a `global *` declaration. Golua allows it.

```lua
goto l2
global *
::l1:: ::l2:: print(3)  -- Should error: scope of '*'
```

---

### GOLUA-011: Coroutine close behavior differs
**Test file**: coroutine.lua

Multiple differences in coroutine closing:
- Closing a coroutine within `__close` may hang
- Dead coroutine returning error on subsequent close behaves differently

---

### GOLUA-012: Stack overflow detection differs
**Test file**: coroutine.lua, cstack.lua

Go doesn't detect stack overflow the same way as C Lua. Tests that rely on stack overflow behavior need to be simulated.

---

### GOLUA-013: Yielding allowed in more contexts
**Test file**: coroutine.lua

Golua allows yielding from contexts where reference Lua does not (e.g., `string.gsub` callbacks).

---

### GOLUA-014: `debug.getinfo` differences
**Test file**: coroutine.lua

- `Y.what` returns `"C"` instead of `nil` for some cases
- Missing `linedefined` field

---

### GOLUA-015: Weak table GC timing differs
**Test file**: closure.lua, coroutine.lua

Go's garbage collector doesn't clear weak references at the same timing as Lua's GC. Tests that rely on immediate weak table clearing after `collectgarbage()` may fail.

---

### GOLUA-016: `debug.upvalueid` fails for Go functions
**Test file**: closure.lua

Calling `debug.upvalueid` on built-in functions like `string.gmatch` returns errors because they're Go functions, not Lua closures.

---

### GOLUA-020: Random number generator differs
**Test file**: math.lua

Golua uses Go's random number generator which has different:
- Algorithm (different sequence for same seed)
- Precision characteristics (more bits than Lua's 53-bit floats)

```lua
math.randomseed(1007)
math.random(0)  -- Different value than reference Lua
```

---

### GOLUA-021: `math.random` accepts extra arguments
**Test file**: math.lua

Golua's `math.random` accepts and ignores extra arguments, while Lua errors.

```lua
math.random(1, 2, 3)  -- Lua: error, Golua: returns random in [1,2]
```

---

### GOLUA-022: `tostring` doesn't preserve `.0` suffix
**Test file**: math.lua

Golua's `tostring` for floating-point numbers that are whole numbers doesn't preserve the decimal point.

```lua
tostring(tonumber("698.0"))  -- Lua: "698.0", Golua: "698"
```

---

### GOLUA-031: GC memory counting differs
**Test file**: vararg.lua

Go's garbage collector counts memory differently than Lua's, so `collectgarbage("count")` comparisons may fail.

---

### GOLUA-033: Infinite coroutine creation not detected
**Test file**: coroutine.lua

Lua detects and errors on infinite coroutine creation. Golua doesn't detect this.

```lua
a = function(a) coroutine.wrap(a)(a) end
pcall(a, a)  -- Lua errors, Golua may hang or OOM
```

---

### GOLUA-037: `table.move` iteration order differs
**Test file**: sort.lua

When `table.move` is called with large index ranges that would cause an error during the move, Lua and Golua iterate in different directions. This affects which element is accessed first.

```lua
-- Moving from mininteger to -2, starting at position 0
-- Lua reads mininteger first, writes 0 first
-- Golua reads -2 first, writes maxinteger-1 first
```

---

### GOLUA-039: `table.sort` doesn't detect invalid order functions
**Test file**: sort.lua

Lua detects and errors when a sort comparison function is not a strict weak ordering (i.e., returns true for both `f(a,b)` and `f(b,a)`). Golua doesn't detect this.

```lua
local function f(a, b) return true end  -- always true, invalid ordering
table.sort({1,2,3,4}, f)
-- Lua: error "invalid order function for sorting"
-- Golua: silently succeeds
```

---

### GOLUA-040: File metatable `__name` is "file" not "FILE*"
**Test file**: files.lua

```lua
getmetatable(io.input()).__name
-- Lua: "FILE*"
-- Golua: "file"
```

---

### GOLUA-041: `load()` accepts "B" mode (no fixed buffer concept)
**Test file**: files.lua

Golua's `load()` function accepts the "B" mode without error, unlike reference Lua which rejects it because Lua code cannot use chunks with fixed buffers.

```lua
load("", "", "B")
-- Lua: error "invalid mode"
-- Golua: returns nil (no error about mode)
```

---

### GOLUA-043: `io.lines` accepts unlimited arguments
**Test file**: files.lua

Golua's `io.lines` accepts and ignores extra arguments beyond the limit, while Lua errors.

```lua
local t = {}; for i = 1, 251 do t[i] = 1 end
io.lines(file, table.unpack(t))
-- Lua: error "too many arguments"
-- Golua: silently succeeds
```
