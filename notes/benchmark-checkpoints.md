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

## 2026-11-14, golua over its revisions, Apple M1 Max

Taken with [`benchmarks/gauge.sh`](../benchmarks/gauge.sh), 3 rounds,
against C Lua 5.5.1 (Homebrew), all four built the same day with Go 1.27.4:
the previous checkpoint (f3b177e, #235), lua5.5 after the work on loops and
conditions (b2af912: #245 to #248, #251, #252), after the run loop lost its
bounds checks (8775256: #253), and lua5.5 built with a profile (00791a0:
#254, #255).

| benchmark | f3b177e | b2af912 | 8775256 | 00791a0 |
|---|---|---|---|---|
| bounce | 2.61× | 2.39× | 2.30× | 2.09× |
| cd | 1.86× | 1.80× | 1.77× | 1.59× |
| deltablue | 2.45× | 2.38× | 2.34× | 2.16× |
| havlak | 1.70× | 1.66× | 1.64× | 1.52× |
| json | 2.59× | 2.52× | 2.46× | 2.21× |
| list | 2.10× | 2.08× | 1.99× | 1.89× |
| mandelbrot | 1.92× | 1.72× | 1.60× | 1.42× |
| nbody | 2.34× | 2.23× | 2.14× | 1.87× |
| permute | 2.65× | 2.48× | 2.39× | 2.26× |
| queens | 2.54× | 2.41× | 2.33× | 2.26× |
| richards | 2.45× | 2.41× | 2.31× | 2.15× |
| sieve | 2.82× | 2.34× | 2.17× | 2.14× |
| storage | 1.83× | 1.76× | 1.74× | 1.65× |
| towers | 2.98× | 2.92× | 2.81× | 2.67× |
| binary-trees | 2.56× | 2.50× | 2.48× | 2.42× |
| fannkuch-redux | 2.18× | 1.76× | 1.70× | 1.67× |
| spectral-norm | 2.49× | 2.33× | 2.17× | 2.07× |
| fasta | 2.72× | 2.45× | 2.37× | 2.28× |
| k-nucleotide | 1.72× | 1.65× | 1.62× | 1.45× |
| regex-dna | 1.31× | 1.31× | 1.31× | 1.26× |
| coroutine-ring | 5.07× | 4.93× | 4.89× | 4.60× |
| md5 | 2.69× | 2.64× | 2.48× | 2.34× |
| life | 1.41× | 1.30× | 1.27× | 1.26× |
| **geomean of 23** | **2.30×** | **2.17×** | **2.10×** | **1.97×** |

Notes:

- golua is below 2× C Lua for the first time: 1.97×, from 2.34× three days
  ago, 16% faster.  The four steps are of four different kinds: the Go
  toolchain, the code the compiler generates, the run loop's fixed costs,
  and how golua itself is compiled.
- The previous checkpoint read 2.34× for f3b177e and this one reads 2.30×:
  that is Go 1.25.4 to 1.27.4.  It is not a uniform gain.  What allocates
  is faster, from the new collector (binary-trees 2.94× to 2.56×,
  coroutine-ring 5.52× to 5.07×, k-nucleotide 1.88× to 1.72×, deltablue
  2.58× to 2.45×, havlak 1.81× to 1.70×); loops that only compute are a
  little slower (sieve 2.65× to 2.82×, md5 2.56× to 2.69×, spectral-norm
  2.38× to 2.49×).
- f3b177e to b2af912 is the generated code: a numeric for loop's body reads
  the counter instead of a copy of it (#245, #246), a for loop opcode takes
  the jump that follows it (#247), a comparison used as a condition is one
  opcode that decides its jump, with no register for the boolean (#248),
  and generic for loops and repeat loops test their conditions the same way
  (#251).  Executed opcodes fall by about 12%.  The loop benchmarks gain
  most: fannkuch 2.18× to 1.76×, sieve 2.82× to 2.34×, mandelbrot 1.92× to
  1.72×, fasta 2.72× to 2.45×, bounce 2.61× to 2.39×.  The call-heavy ones
  hardly move (towers 2.98× to 2.92×, richards 2.45× to 2.41×).  #252, in
  the same step, makes the resource accounting calls of the build without
  quotas compile to nothing, which they did not quite.
- b2af912 to 8775256 is #253: without quotas, the run loop reads
  registers, opcodes, constants and lookup hints without bounds checks,
  relying on the verifier.  3% across the board, most where opcodes are
  cheapest (mandelbrot 1.72× to 1.60×, sieve 2.34× to 2.17×, spectral-norm
  2.33× to 2.17×, md5 2.64× to 2.48×).  Timed alone, the checks on
  registers, on opcodes and on constants were each worth next to nothing;
  the gain appears when all are gone.
- 8775256 to 00791a0 changes no Go code: it is the profile-guided build
  (#254).  The Go compiler reads `default.pgo`, a CPU profile of golua
  running these benchmarks, inlines the calls it shows are hot and turns
  hot interface calls into direct ones.  6%, the largest single step:
  mandelbrot 1.60× to 1.42×, nbody 2.14× to 1.87×, json 2.46× to 2.21×, cd
  1.77× to 1.59×, k-nucleotide 1.62× to 1.45×, bounce 2.30× to 2.09×.  It
  is not the benchmarks being learnt: a profile made from the 14 Are We
  Fast Yet programs alone speeds up the 9 others as much as the full
  profile does ([`benchmarks/README.md`](../benchmarks/README.md#the-profile)).
  `gauge.sh` now builds each revision with its own profile if it has one,
  as a build of its `golua` command would be, so this column is what
  `go install` gives.
- The profile in 00791a0 was made before #253 and not again since
  (`benchmarks/pgo.sh -c` says so): #253 changed the text of the run loop
  but not its lines, so the profile still applies.  It is made again for a
  release.
- regex-dna (1.26×) and life (1.26×) are the closest to C Lua, one because
  its time is in the string library, the other in the collector.
  coroutine-ring stays the outlier at 4.6×: its time is in the Go
  scheduler, a coroutine being a goroutine.
- Where the time goes now, from a CPU profile set against the count of
  executed opcodes (taken at b2af912, before the last two steps): calls are
  13% of the opcodes and 24% of the run loop's time, a third of that in
  making and releasing frames; field access is the most executed kind of
  opcode and costs three times what it does in C Lua, a store into a table
  nearly three times, an append five.  Arithmetic and loops are the closest,
  at less than twice.

## 2026-11-11, golua over its revisions, Apple M1 Max

Taken with [`benchmarks/gauge.sh`](../benchmarks/gauge.sh), 3 rounds,
against C Lua 5.5.1 (Homebrew): the previous checkpoint (6e2ae23, #226),
lua5.5 after the call work (b7d9449: #229, #230, #231), and lua5.5 after the
cells work (f3b177e: #233, #234, #235).

| benchmark | 6e2ae23 | b7d9449 | f3b177e |
|---|---|---|---|
| bounce | 2.85× | 2.82× | 2.58× |
| cd | 2.01× | 1.97× | 1.89× |
| deltablue | 2.81× | 2.72× | 2.58× |
| havlak | 1.96× | 1.89× | 1.81× |
| json | 2.85× | 2.79× | 2.65× |
| list | 2.40× | 2.31× | 2.12× |
| mandelbrot | 2.04× | 2.09× | 1.88× |
| nbody | 2.58× | 2.57× | 2.37× |
| permute | 3.08× | 2.93× | 2.61× |
| queens | 2.95× | 2.83× | 2.54× |
| richards | 2.90× | 2.75× | 2.63× |
| sieve | 2.96× | 3.01× | 2.65× |
| storage | 2.12× | 2.10× | 1.98× |
| towers | 3.47× | 3.31× | 2.97× |
| binary-trees | 3.07× | 3.06× | 2.94× |
| fannkuch-redux | 2.43× | 2.45× | 2.12× |
| spectral-norm | 2.88× | 2.79× | 2.38× |
| fasta | 2.82× | 2.82× | 2.65× |
| k-nucleotide | 1.97× | 1.97× | 1.88× |
| regex-dna | 1.35× | 1.35× | 1.36× |
| coroutine-ring | 5.78× | 5.75× | 5.52× |
| md5 | 3.14× | 2.96× | 2.56× |
| life | 1.49× | 1.52× | 1.36× |
| **geomean of 23** | **2.58×** | **2.53×** | **2.34×** |

Notes:

- Since the 8 November checkpoint the suite is 9.3% faster, 2.58× to 2.34×
  C Lua, and for the first time below 2.5×.  The two steps are of a
  different kind.
- 6e2ae23 to b7d9449 is the call protocol: a call stores its arguments
  straight into the callee's registers (#229), fixed registers (#230), and
  the call opcode making the callee's frame instead of a `cont` opcode
  before it (#231).  The call-heavy benchmarks gain 3 to 6% (towers 3.47×
  to 3.31×, md5 3.14× to 2.96×, permute, richards, queens, list,
  deltablue); the loop benchmarks do not move, or move a percent or two the
  wrong way from code layout (mandelbrot 2.04× to 2.09×, sieve 2.96× to
  3.01×).
- b7d9449 to f3b177e is cells becoming boxes: a captured local lives in a
  box, a value a register holds, reached by `load` and `store`; a fixed
  upvalue is read once when the frame is made; registers have no kind, so
  no register access in the run loop tests a bit first.  This is the step
  that moved the loop benchmarks, which the 8 November note said had
  stopped moving: spectral-norm 2.79× to 2.38×, fannkuch 2.45× to 2.12×,
  sieve 3.01× to 2.65×, mandelbrot 2.09× to 1.88×, life 1.52× to 1.36×, md5
  2.96× to 2.56×.  Their cost is per opcode, and this took a branch out of
  every opcode.  The call-heavy ones gain another 3 to 10% (towers 3.31× to
  2.97×, permute 2.93× to 2.61×, queens 2.83× to 2.54×).
- regex-dna (1.35×) is unchanged by either, as always: its time is in the
  string library.  coroutine-ring stays the outlier at 5.5×.
- The reference matters: a first run of this table against
  `/usr/local/bin/lua`, a Lua 5.5.0 build, read 2.80× for 6e2ae23 where the
  Homebrew 5.5.1 reads 2.58×; the ratios between golua revisions were the
  same.  On this machine `lua` in the path, Homebrew's, is the reference.

## 2026-11-08 (evening), golua over its revisions, Apple M1 Max

Taken with [`benchmarks/gauge.sh`](../benchmarks/gauge.sh), 3 rounds,
against C Lua 5.5.1 (Homebrew): this morning's checkpoint (9d0e7f5), lua5.5
after the field-access work (3bdb01f, #218), and lua5.5 after the frame
set-up work (6e2ae23, #226).

| benchmark | 9d0e7f5 | 3bdb01f | 6e2ae23 |
|---|---|---|---|
| bounce | 3.41× | 3.01× | 2.89× |
| cd | 2.25× | 2.49× | 2.02× |
| deltablue | 3.24× | 3.06× | 2.77× |
| havlak | 2.19× | 2.31× | 1.95× |
| json | 3.39× | 3.16× | 2.82× |
| list | 2.74× | 2.50× | 2.39× |
| mandelbrot | 2.06× | 2.07× | 2.04× |
| nbody | 3.21× | 2.56× | 2.56× |
| permute | 3.83× | 3.17× | 3.03× |
| queens | 3.41× | 3.15× | 2.99× |
| richards | 3.49× | 2.98× | 2.86× |
| sieve | 2.93× | 2.96× | 2.94× |
| storage | 2.30× | 2.23× | 2.10× |
| towers | 4.39× | 3.71× | 3.47× |
| binary-trees | 3.19× | 3.22× | 3.03× |
| fannkuch-redux | 2.39× | 2.43× | 2.44× |
| spectral-norm | 3.06× | 3.03× | 2.88× |
| fasta | 2.96× | 2.94× | 2.82× |
| k-nucleotide | 2.06× | 2.02× | 1.99× |
| regex-dna | 1.36× | 1.36× | 1.36× |
| coroutine-ring | 5.86× | 5.77× | 5.75× |
| md5 | 3.38× | 3.29× | 3.15× |
| life | 1.51× | 1.51× | 1.49× |
| **geomean of 23** | **2.84×** | **2.71×** | **2.57×** |

Notes:

- 9d0e7f5 to 3bdb01f is the field-access work: a constant-key access
  remembers the slot that answered (#212, #215), a method call remembers
  where the method is (#220), a field store writes the slot its key has
  (#221), and the pending-finalizer check is one inline load (#218).
  nbody, towers, permute and richards gain most.  cd and havlak got slower
  at 3bdb01f: #220 gave every metatable an allocated cache, and these
  benchmarks make a metatable per object.
- 3bdb01f to 6e2ae23 fixes that (#223: the absent metamethods are bits in
  padding the hash part had anyway) and cuts the cost of setting up a frame
  (#224, #225, #226): a bare call went from about 51 to 48 ns.  cd and
  havlak are now well below where they were this morning, and the
  call-heavy benchmarks (towers, deltablue, json, richards) gain 4 to 11%.
- The loop benchmarks (mandelbrot, sieve, fannkuch) did not move all day:
  their remaining cost is per opcode, not per call or per access.
  coroutine-ring stays the outlier, as a ring of goroutine hand-offs.

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
