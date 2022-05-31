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
	r.SetEnv(g, "goos", rt.StringValue(runtime.GOOS))
	return cleanup
}

func TestIoLib(t *testing.T) {
	luatesting.RunLuaTestsInDir(t, "lua", setup)
}
