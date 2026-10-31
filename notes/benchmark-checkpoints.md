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
