package luatesting

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arnodel/golua/runtime"
)

// RunSource compiles and runs some source code, outputting to the
// provided io.Writer.
func RunSource(r *runtime.Runtime, source []byte) {
	t := r.MainThread()
	// TODO: use the file name
	clos, err := t.CompileAndLoadLuaChunk("luatest", source, r.GlobalEnv())
	if err != nil {
		fmt.Fprintf(r.Stdout, "!!! parsing: %s", err)
		return
	}
	cerr := runtime.Call(t, runtime.FunctionValue(clos), nil, runtime.NewTerminationWith(nil, 0, false))
	if cerr != nil {
		fmt.Fprintf(r.Stdout, "!!! runtime: %s", cerr)
	}
}

// RunLuaTest runs the lua test code in source, running setup if non-nil
// beforehand (with the Runtime instance that will be used in the test).
func RunLuaTest(source []byte, setup func(*runtime.Runtime) func()) error {
	outputBuf := new(bytes.Buffer)
	r := runtime.New(outputBuf)
	if setup != nil {
		cleanup := setup(r)
		defer cleanup()
	}
	checkers := ExtractLineCheckers(source)
	RunSource(r, source)
	return CheckLines(outputBuf.Bytes(), checkers)
}

func RunLuaTestFile(t *testing.T, path string, setup func(*runtime.Runtime) func()) {
	if filepath.Ext(path) != ".lua" {
		return
	}
	isQuotasTest := strings.HasSuffix(path, ".quotas.lua")
	if isQuotasTest {
		if !runtime.QuotasAvailable {
			t.Skip("Skipping quotas test as build does not enforce quotas")
			return
		}
	}
	t.Run(path, func(t *testing.T) {
		src, err := ioutil.ReadFile(path)
		if err != nil {
			t.Error(err)
			return
		}
		err = RunLuaTest(src, setup)
		if err != nil {
			t.Error(err)
		}
	})
}

// RunLuaTestsInDir runs a test for each .lua file in the directory provided.
func RunLuaTestsInDir(t *testing.T, dirpath string, setup func(*runtime.Runtime) func()) {
	runTest := func(path string, info os.FileInfo, err error) error {
		RunLuaTestFile(t, path, setup)
		return nil
	}
	if err := filepath.Walk(dirpath, runTest); err != nil {
		t.Error(err)
	}
}
