package debuglib

import (
	"github.com/arnodel/golua/lib/packagelib"
	rt "github.com/arnodel/golua/runtime"
)

func Load(r *rt.Runtime) {
	pkg := rt.NewTable()
	rt.SetEnv(r.GlobalEnv(), "debug", pkg)
	_ = packagelib.SavePackage(r.MainThread(), rt.String("debug"), pkg)
}
