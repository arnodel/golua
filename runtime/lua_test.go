package runtime_test

import (
	"testing"

	"github.com/arnodel/golua/lib"
	"github.com/arnodel/golua/luatesting"
)

func TestRuntime(t *testing.T) {
	luatesting.RunLuaTestsInDir(t, "lua", lib.Load)
}
