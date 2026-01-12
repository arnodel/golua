# Table Memory Accounting Issue

## Problem Statement

The current table implementation does not accurately track memory usage when operating under memory quotas. This creates a security gap in quota-restricted execution environments.

## Current Behavior

### What Gets Accounted
- **`Table.Set()` operations**: Returns a fixed 16 bytes per insertion (see [runtime/runtime.go:292-295](../runtime/runtime.go))
- This happens at the API boundary when `Runtime.SetTable()` is called

### What Does NOT Get Accounted
1. **`NewTable()` creation**: Creates empty table with nil pointers, no memory accounting
2. **Internal growth operations**:
   - `array.grow()` ([runtime/hashtable.go:560](../runtime/hashtable.go)) - allocates new slice, copies data
   - `hashTable.grow()` ([runtime/hashtable.go:374](../runtime/hashtable.go)) - doubles hash table size
3. **Table preallocation**: `NewTableWithCapacity()` allocates memory upfront without accounting

### The Gap

The fixed 16-byte accounting per `Set()` is an approximation that assumes:
- Tables start small and grow incrementally
- The overhead of tracking exact memory is too expensive

However, internal growth operations can allocate large chunks of memory silently:
- Array grows in powers of 2 (1 → 2 → 4 → 8 → 16... slots)
- Hash table doubles in size when full
- Each `Value` is 24 bytes on 64-bit systems (2x pointer + 8 bytes)

**Example**: A table growing from 512 to 1024 array slots allocates `512 * 24 = 12,288 bytes` without accounting.

## Impact on `table.create`

The Lua 5.5 `table.create(nseq, nrec)` function is designed to preallocate table capacity for performance. However, this conflicts with memory quotas:

**Without proper accounting**:
```lua
-- With memory quota = 1000 bytes
local t = table.create(10000)  -- Preallocates 240KB, bypasses quota!
```

### Current Workaround

We implemented a **conditional approach** in [lib/tablelib/tablelib.go:507-517](../lib/tablelib/tablelib.go):

```go
hardLimits := t.Runtime.HardLimits()
softLimits := t.Runtime.SoftLimits()
if hardLimits.Memory > 0 || softLimits.Memory > 0 {
    // Memory quotas active - don't preallocate (security safe)
    tbl = rt.NewTable()
} else {
    // No memory quotas - preallocate for performance
    tbl = rt.NewTableWithCapacity(int(nseq), int(nrec))
}
```

This ensures:
- ✅ With quotas: No preallocation, can't bypass limits
- ✅ Without quotas: Preallocation works for performance
- ✅ Security: No new vulnerabilities introduced
- ❌ Performance: Can't use preallocation optimization in quota environments

## Potential Solutions

### Option 1: Account at Growth Points

Modify `array.grow()` and `hashTable.grow()` to account for memory before allocating.

**Pros**:
- More accurate accounting
- Catches all allocations

**Cons**:
- Need access to `Thread` or `Runtime` in growth functions
- Current architecture doesn't pass runtime context to these low-level functions
- Significant refactoring required

**Implementation sketch**:
```go
func (a *array) growWithAccounting(t *Thread, newSize uintptr) error {
    bytesNeeded := uint64(newSize-len(a.values)) * uint64(unsafe.Sizeof(Value{}))
    t.RequireBytes(bytesNeeded)
    // ... perform growth
}
```

### Option 2: Two-Phase Allocation

1. Calculate memory needed for operation
2. Require memory at API boundary
3. Perform operation (growth may still occur, but we pre-accounted)

**Pros**:
- No changes to internal table structure
- Accounts at API boundary (current pattern)

**Cons**:
- Complex to calculate exact growth behavior ahead of time
- May over-account (conservative estimation)
- Doesn't catch all growth scenarios

**Implementation sketch**:
```go
func (r *Runtime) SetTable(t *Table, k, v Value) {
    r.RequireCPU(1)

    // Calculate potential growth
    bytesNeeded := estimateMemoryForSet(t, k, v)
    r.RequireMem(bytesNeeded)

    t.Set(k, v)
}
```

### Option 3: Capacity vs. Used Memory Tracking

Track both capacity and used memory separately:
- Capacity: Total allocated memory
- Used: Memory for non-nil values

**Pros**:
- Accurate accounting of actual allocations
- Can handle preallocation correctly

**Cons**:
- Requires tracking metadata in table structures
- Performance overhead for tracking
- Complexity in implementation

### Option 4: Accept Current Approximation

Document the limitation and keep the fixed 16-byte accounting.

**Pros**:
- Simple, no changes needed
- Performance overhead is minimal
- Works well for typical use cases

**Cons**:
- Security gap remains
- Can't enable preallocation with quotas

## Recommendation

**Short term**: Keep current workaround (conditional preallocation)
- Maintains security in quota environments
- No performance regression for non-quota use cases
- Simple and maintainable

**Long term**: Investigate Option 1 or Option 3
- Option 1: Best for security and accuracy
  - Requires architectural changes to pass runtime context
  - Consider making `mixedTable` aware of quotas
- Option 3: Best for enabling preallocation with quotas
  - Track capacity separately from used memory
  - Account for capacity growth at allocation points

## Action Items

When implementing comprehensive table memory accounting:

1. ✅ **Verify accounting accuracy**
   - Add tests for memory consumption during growth
   - Test edge cases (power-of-2 boundaries, hash collisions)

2. ✅ **Revisit `table.create` preallocation**
   - Once proper accounting exists, we can preallocate even with quotas
   - Update [lib/tablelib/tablelib.go:507-517](../lib/tablelib/tablelib.go) to always use `NewTableWithCapacity`
   - Remove conditional logic

3. ✅ **Update memory quota tests**
   - Ensure tests verify actual memory consumption
   - Test that quotas are enforced during growth

4. ✅ **Performance benchmarks**
   - Measure overhead of accurate accounting
   - Compare with current 16-byte approximation
   - Ensure no regression in common table operations

## References

- Original discovery: During `table.create` implementation (2026-01-11)
- Related code:
  - [runtime/hashtable.go](../runtime/hashtable.go) - `array.grow()`, `hashTable.grow()`
  - [runtime/runtime.go:292-295](../runtime/runtime.go) - Current `SetTable()` accounting
  - [lib/tablelib/tablelib.go:479-520](../lib/tablelib/tablelib.go) - `table.create` implementation
- Lua 5.5 feature: [notes/lua5.5-features.md](lua5.5-features.md)
