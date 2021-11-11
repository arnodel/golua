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
	r.SetEnvGoFunc(pkg, "memory", getMemQuota, 0, false)
	r.SetEnvGoFunc(pkg, "cpu", getCPUQuota, 0, false)
	return rt.TableValue(pkg)
}

func getMemQuota(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	used, max := t.MemQuotaStatus()
	return c.PushingNext(
		rt.IntValue(int64(used)),
		rt.IntValue(int64(max)),
	), nil
}

func getCPUQuota(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	used, max := t.CPUQuotaStatus()
	return c.PushingNext(
		rt.IntValue(int64(used)),
		rt.IntValue(int64(max)),
	), nil
}
