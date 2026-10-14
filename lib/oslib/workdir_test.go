package oslib_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/arnodel/golua/lib"
	rt "github.com/arnodel/golua/runtime"
)

// The working directory of a context applies to file names and to commands, in
// both the quotas and noquotas builds.
func TestContextWorkDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "marker.txt"), []byte("marked"), 0o644); err != nil {
		t.Fatal(err)
	}
	listCmd := "ls"
	if runtime.GOOS == "windows" {
		listCmd = "dir /b"
	}

	var out bytes.Buffer
	r := rt.New(&out)
	defer lib.LoadAll(r)()
	th := r.MainThread()
	src := `
		local listCmd = ...
		local f = assert(io.open("marker.txt"))
		print(f:read("a"))
		f:close()
		local p = assert(io.popen(listCmd))
		print("popen", p:read("a"))
		p:close()
		os.execute(listCmd)
	`
	chunk, err := r.CompileAndLoadLuaChunk("test", []byte(src), rt.TableValue(r.GlobalEnv()))
	if err != nil {
		t.Fatal(err)
	}
	_, err = th.CallContext(rt.RuntimeContextDef{WorkDir: dir}, func() error {
		return rt.Call(th, rt.FunctionValue(chunk), []rt.Value{rt.StringValue(listCmd)}, rt.NewTerminationWith(nil, 0, false))
	})
	if err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "marked") {
		t.Errorf("io.open did not use the working directory: %q", got)
	}
	if strings.Count(got, "marker.txt") != 2 {
		t.Errorf("io.popen and os.execute should both list marker.txt: %q", got)
	}
}
