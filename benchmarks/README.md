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

The default build keeps account of CPU and memory use so that quotas can be
enforced, which costs a few percent.  Performance work targets the build
without it; add `-tags noquotas` to time that one:

```sh
go test -tags noquotas -run '^$' -bench . -count 6 ./benchmarks/ > new.txt
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

[`clbg/`](clbg) holds seven of the
[Benchmarks Game](https://benchmarksgame-team.pages.debian.net/benchmarksgame/)'s
Lua programs.  They are under the Benchmarks Game's revised BSD license,
[`clbg/LICENSE`](clbg/LICENSE).

Three of them, by Mike Pall, are numeric and Are We Fast Yet lacks them.
They come from [gligneul/Lua-Benchmarks](https://github.com/gligneul/Lua-Benchmarks)
by way of [apogee](https://github.com/matjam/apogee), which changed each to
return a function computing its result in place of printing it.

| Benchmark | What it does |
|---|---|
| binary-trees | allocating and walking binary trees, depth 12; the garbage collector |
| fannkuch-redux | permutations of 9 elements in arrays |
| spectral-norm | the spectral norm of a 200×200 matrix; calls and arithmetic |

The other four cover what the rest of the suite leaves out (see
[Coverage](#coverage)).  fasta, k-nucleotide and regex-dna come from
gligneul/Lua-Benchmarks, and coroutine-ring, the Benchmarks Game's
thread-ring, from [LuaJIT-test-cleanup](https://github.com/LuaJIT/LuaJIT-test-cleanup).
Each was changed to return a function computing its output, in place of
reading stdin and writing stdout; each file's header says how.

| Benchmark | What it does |
|---|---|
| fasta | generating 250 KB of DNA sequences; `%`, string.sub, string.char with 2047 arguments, varargs, load |
| k-nucleotide | counting the substrings of a sequence; string keys, an `__index` metamethod, table.sort with a comparison function |
| regex-dna | matching and substituting patterns in a sequence; the string library's pattern matching |
| coroutine-ring | passing a token around a ring of 503 coroutines 100,000 times; coroutine.resume and coroutine.yield |

k-nucleotide and regex-dna read the output of fasta, which the test computes
once before timing them.

## Other programs

[`extra/`](extra) holds two programs from neither suite.

| Benchmark | What it does | From |
|---|---|---|
| md5 | hashing 16 KB of text 20 times; the bitwise operators | md5test.lua from [Lua BitOp](https://bitop.luajit.org/) 1.0.3, public domain |
| life | 200 generations of Conway's Game of Life on 40×20 cells; building strings by concatenation, global variables | test/life.lua from Lua 5.1.5, MIT |

md5 used the bit library, which Lua 5.3 replaced with operators.  It now
uses the operators, and still passes the original's test vectors.  life drew
each generation on a terminal; it now returns the last one.

## Coverage

Are We Fast Yet keeps to features that all its languages share: objects,
closures, arrays and maps.  So it leaves out most of what is particular to
Lua, and the three numeric Benchmarks Game programs add little of it.
Counting the opcodes each benchmark executes showed that the first 17
benchmarks never run string concatenation or floor division, hardly ever
`^`, and run the bitwise operators and `%` at under 2% of their opcodes.
They rarely call the string library, and never resume a coroutine.  A
change that made those slower would not show.

The last six benchmarks fill those gaps:

| | in the first 17, at most | in the new ones |
|---|---|---|
| bitwise operators | 1.7% of opcodes (storage) | 21% (md5) |
| `%` | 1.7% (storage) | 3.1% (fasta) |
| concatenation | never | 1.7% (life), with strings growing to 820 bytes |
| coroutines | never | 100,000 switches each way (coroutine-ring) |
| string library | rarely | 70% of regex-dna's time, in pattern matching; string.sub on every step of k-nucleotide |

Floor division, `^` and error handling with `pcall` are still not covered.

## Sizes

The first 17 benchmarks run at the sizes apogee uses, so results can be set
beside its own: cd with 10 aircraft, deltablue with chains of 1000 variables,
havlak at its smallest size, mandelbrot at 500×500, and one inner iteration
for the others.  The last six have sizes chosen to take 10 to 50
milliseconds in PUC Lua.
