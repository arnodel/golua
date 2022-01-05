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
	"github.com/arnodel/golua/lib/runtimelib"
	"github.com/arnodel/golua/lib/stringlib"
	"github.com/arnodel/golua/lib/tablelib"
	"github.com/arnodel/golua/lib/utf8lib"
	rt "github.com/arnodel/golua/runtime"
)

func LoadLibs(r *rt.Runtime, loaders ...packagelib.Loader) func() {
	var cleanups []func()
	for _, loader := range loaders {
		cleanup := loader.Run(r)
		if cleanup != nil {
			cleanups = append(cleanups, cleanup)
		}
	}
	return func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}
}

func LoadAll(r *rt.Runtime) func() {
	return LoadLibs(
		r,
		base.LibLoader,
		packagelib.LibLoader,
		coroutine.LibLoader,
		stringlib.LibLoader,
		tablelib.LibLoader,
		mathlib.LibLoader,
		iolib.LibLoader,
		utf8lib.LibLoader,
		oslib.LibLoader,
		debuglib.LibLoader,
		golib.LibLoader,
		runtimelib.LibLoader,
	)
}
