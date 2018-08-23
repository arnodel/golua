package mathlib_test

import (
	"testing"

	"github.com/arnodel/golua/lib"
	"github.com/arnodel/golua/luatesting"
)

func TestMathLib(t *testing.T) {
	luatesting.RunLuaTestsInDir(t, "lua", lib.Load)
}
