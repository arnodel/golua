package runtime_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	rt "github.com/arnodel/golua/runtime"
)

// FuzzCompile compiles any input as a Lua chunk.  It may be rejected with an
// error, but it must not crash the scanner, the parser or the compiler, every
// function the compiler emits must pass the bytecode verifier (loading a
// compiled unit panics if one doesn't), and compiling the same source twice
// must give the same code.
//
// The seeds are the Lua files of the repository.  To fuzz:
//
//	go test -run '^$' -fuzz FuzzCompile ./runtime
func FuzzCompile(f *testing.F) {
	for _, pattern := range []string{"lua/*.lua", "../lib/*/lua/*.lua", "../benchmarks/*/*.lua"} {
		paths, err := filepath.Glob(pattern)
		if err != nil {
			f.Fatal(err)
		}
		for _, path := range paths {
			src, err := os.ReadFile(path)
			if err != nil {
				f.Fatal(err)
			}
			f.Add(src)
		}
	}
	f.Fuzz(func(t *testing.T, src []byte) {
		r := rt.New(nil)
		unit, _, err := r.CompileLuaChunk("fuzz", src)
		if err != nil {
			return
		}
		r.LoadLuaUnit(unit, rt.TableValue(r.GlobalEnv()))
		again, _, err := r.CompileLuaChunk("fuzz", src)
		if err != nil {
			t.Fatalf("compiles once, then fails: %s", err)
		}
		if !reflect.DeepEqual(unit, again) {
			t.Fatal("compiling twice gives different code")
		}
	})
}
