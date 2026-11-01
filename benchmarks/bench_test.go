// Package benchmarks measures golua on standard Lua benchmarks, so that its
// performance can be tracked over time and compared with other Lua
// implementations.  See README.md.
package benchmarks

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/arnodel/golua/lib"
	rt "github.com/arnodel/golua/runtime"
)

// A benchmark is Lua source that defines a global function run.  Each call of
// run does one iteration of the benchmark and returns its result as a string,
// which must be want.  A want of the form "sha256:HEX" stands for a result
// too long to spell out, whose SHA-256 is HEX.
type benchmark struct {
	name, src, want string
}

// The 14 benchmarks of Are We Fast Yet and the three numeric Benchmarks Game
// programs it lacks, at the sizes apogee uses (github.com/matjam/apogee,
// bench/), so results can be set beside theirs.  Then programs for what those
// leave out (see "Coverage" in README.md).
var benchmarks = []benchmark{
	awfy("bounce", 1),
	awfy("cd", 10),
	awfy("deltablue", 1000),
	awfy("havlak", 1),
	awfy("json", 1),
	awfy("list", 1),
	awfy("mandelbrot", 500),
	// nbody only knows its expected result after 1 or 250,000 steps, so it
	// reports its energy after 1000 steps and want is what PUC Lua computes.
	{"nbody", `package.path = "awfy/?.lua"
		local b = require "nbody"
		local energy
		function b:verify_result(e) energy = e return true end
		function run() assert(b:inner_benchmark_loop(1000)) return string.format("%.12f", energy) end`,
		"-0.169087605235"},
	awfy("permute", 1),
	awfy("queens", 1),
	awfy("richards", 1),
	awfy("sieve", 1),
	awfy("storage", 1),
	awfy("towers", 1),
	clbg("binary-trees", "return tostring(f(12))", "-2.0"),
	clbg("fannkuch-redux", "local sum, flips = f(9) return tostring(sum * 100 + flips)", "862930"),
	clbg("spectral-norm", "return string.format('%.12f', f(200))", "1.274223601353"),

	// Programs for what the ones above leave out: string processing and the
	// string library, string building by concatenation, bitwise operators,
	// coroutines, metamethods, varargs and global variables.  Their sizes are
	// chosen to take 10 to 50 milliseconds in PUC Lua.
	clbg("fasta", "return f(25000)", "sha256:e1c2e901448dbe22bbc4e85535acf3b2052153c6dc6208c77f54a6e00cf3e91d"),
	withFasta("k-nucleotide", 10000, "sha256:efb8b49cf22ed4a578ff65926943cbec607e69a7bba91a4666b953e92c1db39f"),
	withFasta("regex-dna", 25000, "sha256:8ad0871d2aab9de3ca414ff503e0607e449505f0776c956cfafd134712a65340"),
	clbg("coroutine-ring", "return tostring(f(100000))", "407"),
	extra("md5", "return f(20)", "a831e91e0f70eddcb70dc61c6f82f6cd"),
	extra("life", "return f(40, 20, 200)", "sha256:ea614de27597fdca26940414b93eef6eab93648115cd15b831bf8794505cbacc"),
}

// awfy is an Are We Fast Yet benchmark, run with the given inner iterations
// (the problem size for some).  It checks its own result.
func awfy(name string, inner int) benchmark {
	return benchmark{name, fmt.Sprintf(`package.path = "awfy/?.lua"
		local b = require %q
		function run() assert(b:inner_benchmark_loop(%d), "wrong result") return "ok" end`, name, inner), "ok"}
}

// clbg is a Benchmarks Game program.  Its file returns a function, which body
// calls f.
func clbg(name, body, want string) benchmark {
	return benchmark{name, fmt.Sprintf(`package.path = "clbg/?.lua"
		local f = require %q
		function run() %s end`, name, body), want}
}

// withFasta is a Benchmarks Game program whose input is the output of fasta
// for n, which it computes once before the iterations.  Its file returns a
// function of the input that returns the output.
func withFasta(name string, n int, want string) benchmark {
	return benchmark{name, fmt.Sprintf(`package.path = "clbg/?.lua"
		local input = require "fasta"(%d)
		local f = require %q
		function run() return f(input) end`, n, name), want}
}

// extra is a program from outside the two suites, in extra/.  Its file
// returns a function, which body calls f.
func extra(name, body, want string) benchmark {
	return benchmark{name, fmt.Sprintf(`package.path = "extra/?.lua"
		local f = require %q
		function run() %s end`, name, body), want}
}

// matches reports whether got is the result want describes.
func matches(got, want string) bool {
	if hex, ok := strings.CutPrefix(want, "sha256:"); ok {
		return fmt.Sprintf("%x", sha256.Sum256([]byte(got))) == hex
	}
	return got == want
}

// newGolua returns a golua runtime in which bm.src has run, and its run
// function.
func newGolua(tb testing.TB, bm benchmark) (*rt.Runtime, rt.Value) {
	tb.Helper()
	r := rt.New(io.Discard)
	tb.Cleanup(lib.LoadAll(r))
	// golua's _VERSION is "Golua 5.5", which sorts before "Lua 5.3".  Are We
	// Fast Yet compares _VERSION with "Lua 5.3" to choose between bitwise
	// operators and the bit32 library, which golua lacks.
	r.SetEnv(r.GlobalEnv(), "_VERSION", rt.StringValue("Lua 5.5"))
	chunk, err := r.CompileAndLoadLuaChunk(bm.name, []byte(bm.src), rt.TableValue(r.GlobalEnv()))
	if err != nil {
		tb.Fatal(err)
	}
	if _, err := rt.Call1(r.MainThread(), rt.FunctionValue(chunk)); err != nil {
		tb.Fatal(err)
	}
	return r, r.GlobalEnv().Get(rt.StringValue("run"))
}

// runGolua runs one iteration and returns its result.
func runGolua(tb testing.TB, r *rt.Runtime, run rt.Value) string {
	v, err := rt.Call1(r.MainThread(), run)
	if err != nil {
		tb.Fatal(err)
	}
	s, _ := v.ToString()
	return s
}

// refLua is the reference Lua interpreter to compare with, if
// GOLUA_BENCH_LUA names one.
var refLua = os.Getenv("GOLUA_BENCH_LUA")

// runLua runs bm.src followed by tail in refLua and returns what it writes.
func runLua(tb testing.TB, bm benchmark, tail string) string {
	tb.Helper()
	cmd := exec.Command(refLua, "-")
	cmd.Stdin = strings.NewReader(bm.src + "\n" + tail)
	out, err := cmd.CombinedOutput()
	if err != nil {
		tb.Fatalf("%s: %v\n%s", refLua, err, out)
	}
	return string(out)
}

// describe quotes a result, or gives its length and SHA-256 if it is long.
func describe(got string) string {
	if len(got) <= 100 {
		return fmt.Sprintf("%q", got)
	}
	return fmt.Sprintf("%d bytes with sha256:%x", len(got), sha256.Sum256([]byte(got)))
}

// Each benchmark computes its expected result in golua, and in the reference
// Lua if there is one.
func TestBenchmarks(t *testing.T) {
	if testing.Short() {
		t.Skip("benchmarks take a while")
	}
	for _, bm := range benchmarks {
		t.Run(bm.name, func(t *testing.T) {
			r, run := newGolua(t, bm)
			if got := runGolua(t, r, run); !matches(got, bm.want) {
				t.Errorf("golua: got %s, want %q", describe(got), bm.want)
			}
			if refLua != "" {
				if got := runLua(t, bm, "io.write(run())"); !matches(got, bm.want) {
					t.Errorf("%s: got %s, want %q", refLua, describe(got), bm.want)
				}
			}
		})
	}
}

// BenchmarkStandard times one iteration of each benchmark in golua, and in
// the reference Lua if there is one.  The reference Lua's time includes
// starting it and loading the benchmark, spread over b.N iterations.
func BenchmarkStandard(b *testing.B) {
	for _, bm := range benchmarks {
		b.Run(bm.name+"/impl=golua", func(b *testing.B) {
			r, run := newGolua(b, bm)
			for b.Loop() {
				runGolua(b, r, run)
			}
		})
		if refLua != "" {
			b.Run(bm.name+"/impl=lua", func(b *testing.B) {
				runLua(b, bm, fmt.Sprintf("for _ = 1, %d do run() end", b.N))
			})
		}
	}
}
