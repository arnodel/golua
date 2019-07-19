package golib_test

import (
	"testing"

	"github.com/arnodel/golua/lib/golib"
	"github.com/arnodel/golua/luatesting"
	rt "github.com/arnodel/golua/runtime"
)

func setup(r *rt.Runtime) {
	g := r.GlobalEnv()
	rt.SetEnv(g, "hello", rt.String("world"))
	rt.SetEnv(g, "foo", golib.NewGoValue(r, func(x int) int { return 2 * x }))
}

func TestGoLib(t *testing.T) {
	luatesting.RunLuaTestsInDir(t, "lua", setup)
}
