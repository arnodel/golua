package quotalib

import (
	"github.com/arnodel/golua/lib/packagelib"
	rt "github.com/arnodel/golua/runtime"
)

var LibLoader = packagelib.Loader{
	Load: load,
	Name: "quota",
}

func load(r *rt.Runtime) rt.Value {
	pkg := rt.NewTable()
	r.SetEnvGoFunc(pkg, "get", getQuota, 0, false)
	return rt.TableValue(pkg)
}

func getQuota(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	return c.PushingNext(
		rt.IntValue(int64(t.CurrentCPUQuota())),
		rt.IntValue(int64(t.CurrentMemQuota())),
	), nil
}
