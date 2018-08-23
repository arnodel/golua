package utf8lib_test

import (
	"testing"

	"github.com/arnodel/golua/lib"
	"github.com/arnodel/golua/luatesting"
)

func TestUtf8Lib(t *testing.T) {
	luatesting.RunLuaTestsInDir(t, "lua", lib.Load)
}
