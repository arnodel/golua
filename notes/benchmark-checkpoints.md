# Benchmark checkpoints

Where golua stands against other Lua implementations on the benchmark suite
([`benchmarks/`](../benchmarks)), recorded now and then so the next checkpoint
is comparable.  Each table is one machine, one day, best of three runs per
benchmark, every implementation relative to a C Lua reference (below 1× is
faster than C Lua).

## How to take one

The harness runs any command-line interpreter that takes a program on stdin
with `-`, through `GOLUA_BENCH_LUA`, alongside golua, and checks both compute
the same results:

```sh
go test -c -tags noquotas -o /tmp/bench.test ./benchmarks
cd benchmarks
for round in 1 2 3; do
  GOLUA_BENCH_LUA=/path/to/interpreter /tmp/bench.test -test.run '^$' -test.bench . -test.count 1
done > /tmp/interp.txt
```

Take the best `ns/op` of the three for each benchmark and implementation.
Every run contains golua, so implementations from different runs are set on a
common scale through golua's time in each run: `impl / lua = (impl / golua in
its run) × (golua / lua in the reference run)`.  Use the noquotas build of
golua and keep one reference build of C Lua: two builds of the same Lua
version can differ by 20 to 30% on single benchmarks from compiler flags
alone (see the 5.5.0 column below).

Implementations used so far: C Lua (`lua`), apogee
(`go install github.com/matjam/apogee/cmd/apogee@latest`; `APOGEE_JIT=off`
for its interpreter alone).  Shopify/go-lua has no command-line interpreter
and implements Lua 5.2, which the benchmarks outgrow; apogee's own harness
drives it as a library and publishes 4.8× C Lua for it.

## 2026-11-08, golua over its revisions, Apple M1 Max

Taken with [`benchmarks/gauge.sh`](../benchmarks/gauge.sh), 3 rounds,
against C Lua 5.5.1 (Homebrew): the previous checkpoint's last column
(94f130b, #186) and lua5.5 after #210.  All 23 benchmarks run on both, so
the geomean is over 23, not 20 as in the table below; the three that join
(fasta, k-nucleotide, regex-dna) are among the closer ones to C Lua, which is
why 94f130b reads 2.99× here and 3.16× there.

| benchmark | 94f130b (3 Nov) | 9d0e7f5 (lua5.5) |
|---|---|---|
| bounce | 3.47× | 3.35× |
| cd | 2.30× | 2.25× |
| deltablue | 3.31× | 3.26× |
| havlak | 2.19× | 2.15× |
| json | 3.43× | 3.38× |
| list | 2.81× | 2.74× |
| mandelbrot | 2.50× | 2.05× |
| nbody | 3.34× | 3.36× |
| permute | 3.94× | 3.73× |
| queens | 3.61× | 3.40× |
| richards | 3.51× | 3.46× |
| sieve | 3.54× | 2.93× |
| storage | 2.32× | 2.28× |
| towers | 4.57× | 4.39× |
| binary-trees | 3.28× | 3.24× |
| fannkuch-redux | 2.92× | 2.39× |
| spectral-norm | 3.20× | 3.10× |
| fasta | 3.19× | 2.95× |
| k-nucleotide | 2.15× | 2.11× |
| regex-dna | 1.36× | 1.36× |
| coroutine-ring | 5.99× | 5.83× |
| md5 | 3.60× | 3.38× |
| life | 1.65× | 1.51× |
| **geomean of 23** | **2.99×** | **2.84×** |

Notes:

- Since 3 November the suite is 5.0% faster, which agrees with the sum of
  the changes measured one by one: hoisting constants out of loops (#203,
  −3.0%), calls listing their arguments (#210, −2.0%), negated literals and
  constant conditions (#198, fannkuch).  The pipeline of IR passes was also
  rebuilt (#200–#209: control-flow graph, register lifetimes computed by the
  code generator, passes as Edits) at no cost in opcodes or time.
- The loop benchmarks moved most: mandelbrot 2.50× to 2.05×, sieve 3.54× to
  2.93×, fannkuch 2.92× to 2.39×; the call-heavy ones a few percent each.
- Time per dispatched opcode on lua5.5 today: 5.6 to 7.5 ns in the loop
  benchmarks, 14 to 23 ns where calls or tables dominate.  The remaining
  gap to C Lua is per-opcode cost in the former and per-call and
  per-allocation cost in the latter, not opcode counts.

## 2026-11-03, golua over its revisions, Apple M1 Max

Taken with [`benchmarks/gauge.sh`](../benchmarks/gauge.sh), 3 rounds,
against C Lua 5.5.1 (Homebrew): the suite's first commit (ed8ba02), the
previous checkpoint (17a888f), before #182 (2b09206), lua5.5 after #182
(aae169b) and #186 (value-andor).  fasta, k-nucleotide and regex-dna fail on
the first two, from golua bugs fixed since (fasta: `too many values to
unpack`), so the geomean is over the other 20.

| benchmark | ed8ba02 | 17a888f | 2b09206 | lua5.5 | value-andor |
|---|---|---|---|---|---|
| bounce | 4.97× | 3.54× | 3.56× | 3.52× | 3.51× |
| cd | 3.07× | 2.31× | 2.31× | 2.29× | 2.28× |
| deltablue | 5.18× | 3.33× | 3.30× | 3.29× | 3.30× |
| havlak | 3.13× | 2.28× | 2.27× | 2.25× | 2.23× |
| json | 5.34× | 3.54× | 3.52× | 3.48× | 3.46× |
| list | 4.82× | 3.18× | 3.20× | 2.84× | 2.82× |
| mandelbrot | 3.07× | 2.70× | 2.71× | 2.50× | 2.51× |
| nbody | 4.75× | 3.26× | 3.30× | 3.23× | 3.24× |
| permute | 6.60× | 4.16× | 4.15× | 4.10× | 4.08× |
| queens | 5.50× | 3.58× | 3.66× | 3.58× | 3.50× |
| richards | 6.63× | 3.66× | 3.69× | 3.63× | 3.60× |
| sieve | 4.56× | 3.65× | 3.66× | 3.52× | 3.53× |
| storage | 2.99× | 2.34× | 2.35× | 2.34× | 2.34× |
| towers | 7.44× | 4.70× | 4.70× | 4.67× | 4.60× |
| binary-trees | 3.63× | 3.23× | 3.25× | 3.24× | 3.24× |
| fannkuch-redux | 4.64× | 3.19× | 3.21× | 2.89× | 2.94× |
| spectral-norm | 3.79× | 3.28× | 3.30× | 3.22× | 3.23× |
| fasta | - | - | 3.25× | 3.19× | 3.21× |
| k-nucleotide | - | - | 2.20× | 2.16× | 2.17× |
| regex-dna | - | - | 1.37× | 1.37× | 1.36× |
| coroutine-ring | 8.22× | 6.21× | 6.21× | 6.06× | 6.01× |
| md5 | 4.40× | 3.62× | 3.65× | 3.63× | 3.64× |
| life | 2.28× | 1.88× | 1.73× | 1.66× | 1.64× |
| **geomean of 20** | **4.51×** | **3.27×** | **3.26×** | **3.17×** | **3.16×** |

Notes:

- From the suite's first commit golua went from 4.51× to 3.16× C Lua on these
  20 benchmarks: 30% less time.
- The previous checkpoint's 3.25× (17 benchmarks) is 3.27× here (20
  benchmarks): the two methods agree.
- #182 (with #183) is the step from 3.26× to 3.17×: list, fannkuch-redux,
  mandelbrot and sieve, the benchmarks whose loops and conditions it shortens.
- Between separate binaries a benchmark can move by a percent or two from
  code layout alone; life gains 1.2% here from #186, where the on/off
  measurement in the same binary gives 3.1%.

## 2026-10-31, golua 17a888f (after #174), Apple M1 Max

Reference: C Lua 5.5.1 (Homebrew).  C Lua 5.5.0 is a separate build of the
previous release, included to show the spread between builds.

| benchmark | C Lua 5.5.0 | apogee 1.1.1 JIT | apogee 1.1.1 interpreter | golua |
|---|---|---|---|---|
| towers | 1.02× | 0.69× | 1.73× | 4.76× |
| permute | 1.03× | 0.52× | 2.02× | 4.22× |
| sieve | 0.82× | 0.51× | 1.92× | 3.69× |
| richards | 0.99× | 0.59× | 1.36× | 3.67× |
| queens | 0.96× | 0.50× | 1.67× | 3.65× |
| bounce | 0.99× | 0.45× | 1.51× | 3.58× |
| json | 1.00× | 0.71× | 1.46× | 3.58× |
| nbody | 0.85× | 0.41× | 1.31× | 3.37× |
| deltablue | 1.00× | 0.82× | 1.30× | 3.37× |
| spectral-norm | 0.78× | 0.40× | 1.44× | 3.28× |
| binary-trees | 0.96× | 0.68× | 1.06× | 3.25× |
| fannkuch-redux | 0.86× | 0.38× | 1.99× | 3.21× |
| list | 0.97× | 0.53× | 1.47× | 3.12× |
| mandelbrot | 0.66× | 0.52× | 1.08× | 2.68× |
| storage | 1.00× | 0.55× | 0.81× | 2.36× |
| cd | 0.97× | 0.59× | 0.93× | 2.33× |
| havlak | 1.00× | 0.56× | 0.82× | 2.24× |
| **geomean** | **0.93×** | **0.54×** | **1.35×** | **3.25×** |

Notes:

- At the suite's first commit (ed8ba02, 2026-10-29) golua was about 4.9× C
  Lua; the interleaved A/B between that commit and this one gives −29%
  geomean on golua's own times.  The gain is in table access and in the
  number of opcodes executed (#151 to #174); arithmetic itself is unchanged.
- golua is 2.4× apogee's interpreter.  The gap is smallest on the
  allocation-heavy benchmarks (storage, cd, havlak: 2.4 to 2.9×) and largest
  on the tight loops (towers, permute, sieve: 2.0 to 2.8× of a larger
  number), which matches golua's profile: the remaining cost is per opcode.
  1.35× is therefore a measured floor for a Go interpreter on this set, not a
  Go ceiling.
- apogee's published figures on its own hardware are 0.57× (JIT), 1.8×
  (interpreter) against C Lua 5.5 on a Ryzen 9, and 0.75× / 1.5× against C
  Lua 5.4 on an M1 Pro.
