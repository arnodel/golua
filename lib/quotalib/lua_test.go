//go:build !noquotas
// +build !noquotas

package quotalib_test

import (
	"testing"

	"github.com/arnodel/golua/lib"
	"github.com/arnodel/golua/luatesting"
	rt "github.com/arnodel/golua/runtime"
)

func TestQuotaLib(t *testing.T) {
	luatesting.RunLuaTestsInDir(t, "lua", func(r *rt.Runtime) func() {
		r.AllowQuotaModificationsInLua()
		return lib.LoadAll(r)
	})
}
