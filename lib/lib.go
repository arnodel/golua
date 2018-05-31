package lib

import (
	"github.com/arnodel/golua/lib/base"
	"github.com/arnodel/golua/lib/coroutine"
	"github.com/arnodel/golua/lib/iolib"
	"github.com/arnodel/golua/lib/mathlib"
	"github.com/arnodel/golua/lib/packagelib"
	"github.com/arnodel/golua/lib/stringlib"
	"github.com/arnodel/golua/lib/tablelib"
	rt "github.com/arnodel/golua/runtime"
)

func Load(r *rt.Runtime) {
	base.Load(r)
	coroutine.Load(r)
	packagelib.Load(r)
	stringlib.Load(r)
	tablelib.Load(r)
	mathlib.Load(r)
	iolib.Load(r)
}
