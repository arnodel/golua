# Benchmarks

Standard Lua benchmarks, to track golua's performance over time and compare
it with other Lua implementations.  [`bench_test.go`](bench_test.go) runs them
with Go's testing package.

## Running them

Check that every benchmark computes its expected result (this takes about 20
seconds, most of it Havlak):

```sh
go test ./benchmarks/
```

Time them:

```sh
go test -run '^$' -bench . -count 6 ./benchmarks/ > new.txt
```

Each benchmark has a sub-benchmark per implementation, named `impl=golua` and
`impl=lua`.  Set `GOLUA_BENCH_LUA` to the path of a reference Lua interpreter
to add `impl=lua`, and to have the test check its results too:

```sh
GOLUA_BENCH_LUA=/usr/local/bin/lua go test -run '^$' -bench . -count 6 ./benchmarks/ > new.txt
```

golua runs in-process, and each iteration is one call of the benchmark.  The
reference Lua runs as a separate process for all of a benchmark's iterations,
so its time includes starting it and loading the benchmark, spread over the
iterations.

To see the effect of a change, compare runs before and after it with
[benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat):

```sh
benchstat old.txt new.txt
```

To set golua beside the reference Lua in one run:

```sh
benchstat -col /impl new.txt
```

## Are We Fast Yet

[`awfy/`](awfy) holds the Lua port of the 14 benchmarks of
[Are We Fast Yet](https://github.com/smarr/are-we-fast-yet) (Marr, Daloze and
Mössenböck, *Cross-Language Compiler Benchmarking: Are We Fast Yet?*, DLS
2016), unchanged from commit 74306fe.  The suite was designed to compare
language implementations on the same object-oriented code, and covers method
calls, closures, table access, strings and allocation.

| Benchmark | What it does |
|---|---|
| bounce | balls bouncing in a box; field access and method calls |
| cd | aircraft collision detection with a red-black tree |
| deltablue | a constraint solver |
| havlak | loop recognition in a large control-flow graph; allocation |
| json | parsing a JSON document |
| list | building and walking linked lists |
| mandelbrot | the Mandelbrot set; floating-point arithmetic |
| nbody | the solar system's planets; floating-point arithmetic on fields |
| permute | generating permutations recursively |
| queens | the eight queens problem |
| richards | an operating system's task scheduler |
| sieve | the sieve of Eratosthenes |
| storage | building a tree of arrays; allocation |
| towers | the towers of Hanoi |

Its own harness is not included, as Go's testing package does the timing.
Each benchmark checks its own result, except nbody, which only knows its
result after 1 or 250,000 steps.  The test runs it for 1000 steps and checks
its energy against what PUC Lua computes.

The suite chooses between bitwise operators and the `bit32` library by
comparing `_VERSION` with `"Lua 5.3"`.  golua's `"Golua 5.5"` sorts before
it, so `bench_test.go` sets `_VERSION` to `"Lua 5.5"` before loading the
benchmarks rather than change the vendored files.

The files keep their license headers: most are MIT, havlak is Apache 2.0,
richards and deltablue carry Mario Wolczko's terms, and nbody comes from the
Benchmarks Game.  See [`awfy/LICENSE.md`](awfy/LICENSE.md) and
[`awfy/AUTORS.md`](awfy/AUTORS.md).

## The Computer Language Benchmarks Game

[`clbg/`](clbg) holds three of the
[Benchmarks Game](https://benchmarksgame-team.pages.debian.net/benchmarksgame/)'s
Lua programs by Mike Pall, which Are We Fast Yet lacks.  They come from
[gligneul/Lua-Benchmarks](https://github.com/gligneul/Lua-Benchmarks) by way
of [apogee](https://github.com/matjam/apogee), which changed each to return a
function computing its result in place of printing it.  They are under the
Benchmarks Game's revised BSD license, [`clbg/LICENSE`](clbg/LICENSE).

| Benchmark | What it does |
|---|---|
| binary-trees | allocating and walking binary trees, depth 12; the garbage collector |
| fannkuch-redux | permutations of 9 elements in arrays |
| spectral-norm | the spectral norm of a 200×200 matrix; calls and arithmetic |

## Sizes

The benchmarks run at the sizes apogee uses, so results can be set beside
its own: cd with 10 aircraft, deltablue with chains of 1000 variables, havlak
at its smallest size, mandelbrot at 500×500, and one inner iteration for the
others.
