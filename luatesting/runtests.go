package luatesting

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/arnodel/golua/lib"
	"github.com/arnodel/golua/runtime"
)

// RunSource compiles and runs some source code, outputting to the
// provided io.Writer.
func RunSource(r *runtime.Runtime, source []byte) {
	t := r.MainThread()
	// TODO: use the file name
	clos, err := runtime.CompileAndLoadLuaChunk("luatest", source, r.GlobalEnv())
	if err != nil {
		fmt.Fprintf(r.Stdout, "!!! parsing: %s", err)
		return
	}
	cerr := runtime.Call(t, runtime.FunctionValue(clos), nil, runtime.NewTerminationWith(0, false))
	if cerr != nil {
		fmt.Fprintf(r.Stdout, "!!! runtime: %s", cerr)
	}
}

// RunLuaTest runs the lua test code in source, running setup if non-nil
// beforehand (with the Runtime instance that will be used in the test).
func RunLuaTest(source []byte, setup func(*runtime.Runtime)) error {
	outputBuf := new(bytes.Buffer)
	r := runtime.New(outputBuf)
	lib.Load(r)
	if setup != nil {
		setup(r)
	}
	checkers := ExtractLineCheckers(source)
	RunSource(r, source)
	return CheckLines(outputBuf.Bytes(), checkers)
}

// RunLuaTestsInDir runs a test for each .lua file in the directory provided.
func RunLuaTestsInDir(t *testing.T, dirpath string, setup func(*runtime.Runtime)) {
	runTest := func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) != ".lua" {
			return nil
		}
		t.Run(path, func(t *testing.T) {
			src, err := ioutil.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			err = RunLuaTest(src, setup)
			if err != nil {
				t.Fatal(err)
			}
		})
		return nil
	}
	filepath.Walk(dirpath, runTest)
}
