package iolib_test

import (
	"testing"

	"github.com/arnodel/golua/lib"
	"github.com/arnodel/golua/luatesting"
)

func TestIoLib(t *testing.T) {
	luatesting.RunLuaTestsInDir(t, "lua", lib.LoadAll)
}
