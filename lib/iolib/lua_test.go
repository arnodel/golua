package iolib_test

import (
	"testing"
	"runtime"

	"github.com/arnodel/golua/lib"
	"github.com/arnodel/golua/luatesting"
	rt "github.com/arnodel/golua/runtime"
)

func setup(r *rt.Runtime) func() {
	cleanup := lib.LoadAll(r)
	g := r.GlobalEnv()
	if runtime.GOOS == "windows" {
		r.SetEnv(g, "readcmd", rt.StringValue("type files/iotest.txt"))
		r.SetEnv(g, "writecmd", rt.StringValue("type con > files/popenwrite.txt"))
	} else {
		r.SetEnv(g, "readcmd", rt.StringValue("cat files/iotest.txt"))
		r.SetEnv(g, "writecmd", rt.StringValue("cat > files/popenwrite.txt"))
	}
	return cleanup
}

func TestIoLib(t *testing.T) {
	luatesting.RunLuaTestsInDir(t, "lua", setup)
}
