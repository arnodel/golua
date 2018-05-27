package tablelib

import rt "github.com/arnodel/golua/runtime"

func Load(r *rt.Runtime) {
	pkg := rt.NewTable()
	rt.SetEnv(r.GlobalEnv(), "table", pkg)
	rt.SetEnvGoFunc(pkg, "concat", concat, 4, false)
	rt.SetEnvGoFunc(pkg, "insert", insert, 3, false)
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
	j, err := rt.Len(t, tbl)
	if err != nil {
		return nil, err.AddContext(c)
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

func insert(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err.AddContext(c)
	}
	tbl, err := c.TableArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	var val rt.Value
	var pos rt.Int
	tblLen, err := rt.Len(t, tbl)
	if err != nil {
		return nil, err.AddContext(c)
	}
	if c.NArgs() >= 3 {
		pos, err = c.IntArg(1)
		if err != nil {
			return nil, err.AddContext(c)
		}
		if pos <= 0 {
			return nil, rt.NewErrorS("#2 out of range").AddContext(c)
		}
		val = c.Arg(2)
	} else {
		pos = tblLen + 1
		val = c.Arg(1)
	}
	var oldVal rt.Value
	for pos <= tblLen {
		oldVal, err = rt.Index(t, tbl, pos)
		if err != nil {
			return nil, err.AddContext(c)
		}
		err = rt.SetIndex(t, tbl, pos, val)
		if err != nil {
			return nil, err.AddContext(c)
		}
		val = oldVal
		pos++
	}
	err = rt.SetIndex(t, tbl, pos, val)
	if err != nil {
		return nil, err.AddContext(c)
	}
	return c.Next(), nil
}
