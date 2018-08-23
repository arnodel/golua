package packagelib_test

import (
	"testing"

	"github.com/arnodel/golua/lib"
	"github.com/arnodel/golua/luatesting"
)

func TestPackageLib(t *testing.T) {
	luatesting.RunLuaTestsInDir(t, "lua", lib.Load)
}
