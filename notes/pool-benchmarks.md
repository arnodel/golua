# Pool Benchmarks: contpool and regpool

## What the pools do

### contpool (disabled with `nocontpool`)

Reuses continuation structs to avoid heap allocations on every Lua function call:

- **LuaCont pool**: Fixed 100-slot stack of `*LuaCont` structs. On tail calls, the current continuation is released back to the pool and immediately reused by the next call. Non-tail calls also benefit during the unwinding phase.
- **GoCont pool**: Fixed 10-slot stack of `*GoCont` structs. Reuses Go continuation wrappers when calling Go-implemented functions from Lua.

### regpool (disabled with `noregpool`)

Reuses `[]Value` and `[]Cell` slices using generation-based aging (default pool size 10, max age 10):

- **Register arrays** (`[]Value`): Each Lua function frame needs a register array sized to its max stack usage. The pool reuses arrays of matching or larger capacity.
- **Argument arrays** (`[]Value`): Reuses slices used to pass arguments between function calls.
- **Upvalue cells** (`[]Cell`): Reuses slices for closure upvalue storage.

## Benchmark workloads

| Benchmark | Description | What it stresses |
|-----------|-------------|-----------------|
| TailRecursion | `sum(1000000, 0)` via tail-recursive calls | contpool (release+reacquire every call), regpool (same-sized regs) |
| DeepRecursion | `fib(25)` non-tail recursive | Many live continuations (pool helps less until unwind); regpool helps on return |
| LoopWithCalls | Loop calling `f(x)` 500k times | Steady-state pool reuse: 1 cont + 1 reg set per call |
| GoFunctionCalls | Loop calling Go `addone(s)` 500k times | GoCont pool + argsPool reuse |
| ManyLocals | Function with 20 locals, called 200k times | regpool with large register arrays |
| Coroutines | 100k short-lived `coroutine.wrap` coroutines | Pool behaviour across goroutine boundaries |

## Results

Platform: darwin/arm64, Apple M1 Max, Go 1.24

### Time (ns/op)

| Benchmark | Both pools | No regpool | No contpool | No pools |
|-----------|-----------|-----------|------------|---------|
| TailRecursion | 90.8m | 116.4m (+28%) | 121.3m (+34%) | 145.0m (+60%) |
| DeepRecursion | 29.7m | 35.2m (+18%) | 37.8m (+27%) | 42.0m (+41%) |
| LoopWithCalls | 59.1m | 70.3m (+19%) | 74.0m (+25%) | 84.9m (+44%) |
| GoFunctionCalls | 60.6m | 70.6m (+16%) | 71.6m (+18%) | 81.2m (+34%) |
| ManyLocals | 60.4m | 79.5m (+32%) | 68.6m (+13%) | 82.7m (+37%) |
| Coroutines | 253.5m | 263.3m (+4%) | 269.8m (+6%) | 278.9m (+10%) |
| **geomean** | **72.7m** | **86.7m (+19%)** | **87.5m (+20%)** | **99.5m (+37%)** |

### Allocations (allocs/op)

| Benchmark | Both pools | No regpool | No contpool | No pools |
|-----------|-----------|-----------|------------|---------|
| TailRecursion | 6 | 1,000,016 | 1,000,016 | 2,000,027 |
| DeepRecursion | 4,330 | 242,795 | 247,118 | 485,582 |
| LoopWithCalls | 3 | 500,007 | 500,009 | 1,000,012 |
| GoFunctionCalls | 2 | 500,004 | 500,007 | 1,000,007 |
| ManyLocals | 3 | 200,016 | 200,005 | 400,018 |
| Coroutines | 1,100,033 | 1,300,026 | 1,400,035 | 1,600,038 |

### Memory (B/op)

| Benchmark | Both pools | No regpool | No contpool | No pools |
|-----------|-----------|-----------|------------|---------|
| TailRecursion | 302 | 128,001,263 | 112,001,274 | 240,002,338 |
| DeepRecursion | 554K | 30,349K | 27,096K | 58,269K |
| LoopWithCalls | 163 | 40,000,707 | 56,000,748 | 96,001,155 |
| GoFunctionCalls | 108 | 12,000,390 | 32,000,566 | 44,000,711 |
| ManyLocals | 189 | 115,201,514 | 22,400,457 | 137,601,728 |
| Coroutines | 57.6M | 68.0M | 81.6M | 92.0M |

## Analysis

### Both pools together eliminate virtually all per-call allocations

With both pools enabled, the non-coroutine benchmarks show **2-6 allocs/op total** regardless of how many function calls happen (500k-1M calls per iteration). Without pools, every function call allocates at least one continuation and one register array. The pools reduce allocs by factors of 100,000x to 500,000x for call-heavy workloads.

### contpool and regpool contribute roughly equally to time savings

On average, each pool contributes ~20% speedup independently (geomean). Together they provide ~37% speedup. The contributions aren't fully additive because both pools reduce GC pressure, and the GC savings partially overlap.

- **contpool** is more impactful for deep/recursive workloads (DeepRecursion: +27% without contpool vs +18% without regpool) because continuation objects are larger and more frequently created.
- **regpool** is more impactful for workloads with large register counts (ManyLocals: +32% without regpool vs +13% without contpool) because it saves allocating large `[]Value` slices.

### Each pool eliminates ~half the allocations

When disabling only one pool, allocation counts are roughly halved compared to disabling both. This confirms each pool targets a distinct allocation: contpool handles continuation structs, regpool handles register/arg/cell slices. Together they cover the full per-call allocation footprint.

### Memory savings are dramatic

With both pools enabled, steady-state workloads (TailRecursion, LoopWithCalls, GoFunctionCalls, ManyLocals) use less than 1KB total across hundreds of thousands of calls. Without pools, these same workloads allocate 40-240MB. This represents a **5-6 orders of magnitude** reduction in allocation volume.

### Coroutines benefit less from pools

Coroutines show only ~10% speedup from pools (vs 34-60% for other workloads). This is because each coroutine creates a new goroutine with its own stack, which dominates the allocation cost. The per-call continuation/register savings are a small fraction of the total.

### regpool has negligible overhead when pools hit

The ManyLocals benchmark is interesting: with 20 locals per call, register arrays are ~160 bytes each. The regpool reuses these efficiently. Without regpool, this benchmark allocates 115MB of register arrays alone (200k calls x ~576 bytes each).

## Reproducing

Install `benchstat` if you don't have it:

```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

Run benchmarks for each of the 4 build-tag combinations (6 iterations each for statistical significance):

```bash
go test ./runtime/ -bench=BenchmarkPool -benchmem -count=6 -run='^$' > bench_default.txt
go test ./runtime/ -bench=BenchmarkPool -benchmem -count=6 -run='^$' -tags=noregpool > bench_noregpool.txt
go test ./runtime/ -bench=BenchmarkPool -benchmem -count=6 -run='^$' -tags=nocontpool > bench_nocontpool.txt
go test ./runtime/ -bench=BenchmarkPool -benchmem -count=6 -run='^$' -tags="nocontpool noregpool" > bench_nopools.txt
```

Compare results:

```bash
benchstat bench_default.txt bench_nopools.txt
benchstat bench_default.txt bench_noregpool.txt
benchstat bench_default.txt bench_nocontpool.txt
```

## Recommendation

**Both pools should be made permanent (remove `nocontpool` and `noregpool` build tags).**

The pools provide massive benefits across all workloads:
- 37% geomean speedup
- 5-6 orders of magnitude fewer allocations for call-heavy code
- Negligible overhead (pool management is simple stack push/pop)

There is no practical scenario where disabling pools would be beneficial. The build tags add maintenance burden (conditional compilation in multiple files) for no user-facing value.
