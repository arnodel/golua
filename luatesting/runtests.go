package luatesting

import (
	"bytes"
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	rt "github.com/arnodel/golua/runtime"
)

// RunSource compiles and runs some source code, outputting to the
// provided io.Writer.
func RunSource(r *rt.Runtime, source []byte) {
	t := r.MainThread()
	// TODO: use the file name
	clos, err := t.LoadFromSourceOrCode("luatest", source, "t", rt.TableValue(r.GlobalEnv()), false)
	if err != nil {
		fmt.Fprintf(r.Stdout, "!!! parsing: %s", err)
		return
	}
	cerr := rt.Call(t, rt.FunctionValue(clos), nil, rt.NewTerminationWith(nil, 0, false))
	if cerr != nil {
		fmt.Fprintf(r.Stdout, "!!! runtime: %s", cerr)
	}
}

// RunLuaTest runs the lua test code in source, running setup if non-nil
// beforehand (with the Runtime instance that will be used in the test).
func RunLuaTest(source []byte, setup func(*rt.Runtime) func()) error {
	outputBuf := new(bytes.Buffer)
	r := rt.New(outputBuf)
	r.SetWarner(rt.NewLogWarner(outputBuf, "Test warning: "))
	if setup != nil {
		cleanup := setup(r)
		defer cleanup()
	}
	checkers := ExtractLineCheckers(source)
	RunSource(r, source)
	r.Close(nil)
	return CheckLines(outputBuf.Bytes(), checkers)
}

func RunLuaTestFile(t *testing.T, path string, setup func(*rt.Runtime) func()) {
	if filepath.Ext(path) != ".lua" {
		return
	}
	isQuotasTest := strings.HasSuffix(path, ".quotas.lua")
	t.Run(path, func(t *testing.T) {
		if isQuotasTest {
			if !rt.QuotasAvailable {
				t.Skip("Skipping quotas test as build does not enforce quotas")
				return
			}
		}
		src, err := ioutil.ReadFile(path)
		if err != nil {
			t.Error(err)
			return
		}

		doRun, err := checkTags(src)
		if !doRun {
			if err != nil {
				t.Error(err)
			} else {
				t.Skip("Skipping because of tags")
			}
			return
		}

		err = RunLuaTest(src, setup)
		if err != nil {
			t.Error(err)
		}
	})
}

// RunLuaTestsInDir runs a test for each .lua file in the directory provided.
func RunLuaTestsInDir(t *testing.T, dirpath string, setup func(*rt.Runtime) func(), filters ...string) {
	runTest := func(path string, entry fs.DirEntry, err error) error {
		for _, filter := range filters {
			if !strings.Contains(entry.Name(), filter) {
				return nil
			}
		}
		RunLuaTestFile(t, path, setup)
		return nil
	}
	if err := filepath.WalkDir(dirpath, runTest); err != nil {
		t.Error(err)
	}
}
