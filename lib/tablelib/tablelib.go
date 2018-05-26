package tablelib

import rt "github.com/arnodel/golua/runtime"

func Load(r *rt.Runtime) {
	pkg := rt.NewTable()
	rt.SetEnv(r.GlobalEnv(), "table", pkg)
	rt.SetEnvGoFunc(pkg, "concat", concat, 4, false)
}

func concat(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	tbl, err := c.TableArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	var sep rt.String
	i := rt.Int(1)
	jj, err := rt.Len(t, tbl)
	if err != nil {
		return nil, err.AddContext(c)
	}
	j, tp := rt.ToInt(jj)
	if tp != rt.IsInt {
		return nil, rt.NewErrorS("table length not an integer").AddContext(c)
	}
Switch:
	switch nargs := c.NArgs(); {
	case nargs >= 4:
		j, err = c.IntArg(3)
		if err != nil {
			break
		}
		fallthrough
	case nargs >= 3:
		i, err = c.IntArg(2)
		if err != nil {
			break
		}
		fallthrough
	case nargs >= 2:
		sep, err = c.StringArg(1)
		if err != nil {
			break
		}
		fallthrough
	default:
		var res rt.Value
		if i > j {
			return c.Next(), nil
		}
		res, err = rt.Index(t, tbl, i)
		if err != nil {
			break
		}
		for i++; i <= j; i++ {
			res, err = rt.Concat(t, res, sep)
			if err != nil {
				break Switch
			}
			v, err := rt.Index(t, tbl, i)
			if err != nil {
				break Switch
			}
			res, err = rt.Concat(t, res, v)
			if err != nil {
				break Switch
			}
		}
		return c.PushingNext(res), nil
	}
	return nil, err.AddContext(c)
}
