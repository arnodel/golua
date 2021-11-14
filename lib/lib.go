package lib

import (
	"github.com/arnodel/golua/lib/base"
	"github.com/arnodel/golua/lib/coroutine"
	"github.com/arnodel/golua/lib/debuglib"
	"github.com/arnodel/golua/lib/golib"
	"github.com/arnodel/golua/lib/iolib"
	"github.com/arnodel/golua/lib/mathlib"
	"github.com/arnodel/golua/lib/oslib"
	"github.com/arnodel/golua/lib/packagelib"
	"github.com/arnodel/golua/lib/quotalib"
	"github.com/arnodel/golua/lib/stringlib"
	"github.com/arnodel/golua/lib/tablelib"
	"github.com/arnodel/golua/lib/utf8lib"
	rt "github.com/arnodel/golua/runtime"
)

func LoadAll(r *rt.Runtime) func() {
	base.Load(r)
	packagelib.LibLoader.Run(r)
	coroutine.LibLoader.Run(r)
	stringlib.LibLoader.Run(r)
	tablelib.LibLoader.Run(r)
	mathlib.LibLoader.Run(r)
	iolib.LibLoader.Run(r)
	utf8lib.LibLoader.Run(r)
	oslib.LibLoader.Run(r)
	debuglib.LibLoader.Run(r)
	golib.LibLoader.Run(r)
	if rt.QuotasAvailable {
		quotalib.LibLoader.Run(r)
	}
	return func() {
		iolib.LibLoader.Cleanup(r)
	}
}
