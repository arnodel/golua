//go:build !noquotas
// +build !noquotas

package quotalib_test

import (
	"testing"

	"github.com/arnodel/golua/lib"
	"github.com/arnodel/golua/luatesting"
)

func TestQuotaLib(t *testing.T) {
	luatesting.RunLuaTestsInDir(t, "lua", lib.LoadAll)
}
