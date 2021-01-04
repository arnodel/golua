package tablelib

import (
	"math"
	"sort"

	"github.com/arnodel/golua/lib/packagelib"

	rt "github.com/arnodel/golua/runtime"
)

// LibLoader can load the table lib.
var LibLoader = packagelib.Loader{
	Load: load,
	Name: "table",
}

func load(r *rt.Runtime) rt.Value {
	pkg := rt.NewTable()
	rt.SetEnvGoFunc(pkg, "concat", concat, 4, false)
	rt.SetEnvGoFunc(pkg, "insert", insert, 3, false)
	rt.SetEnvGoFunc(pkg, "move", move, 5, false)
	rt.SetEnvGoFunc(pkg, "pack", pack, 0, true)
	rt.SetEnvGoFunc(pkg, "remove", remove, 2, false)
	rt.SetEnvGoFunc(pkg, "sort", sortf, 2, false)
	rt.SetEnvGoFunc(pkg, "unpack", unpack, 3, false)
	return rt.TableValue(pkg)
}

func concat(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	_, err := c.TableArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	tblVal := c.Arg(0)
	var (
		sep string
		i   int64 = 1
	)
	j, err := rt.IntLen(t, tblVal)
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
			return c.PushingNext(rt.StringValue("")), nil
		}
		res, err = rt.Index(t, tblVal, rt.IntValue(i))
		if err != nil {
			break
		}
		// log.Printf("CONCAT: j = %d, j = %d", i, j)
		for {
			if i == math.MaxInt64 {
				break
			}
			i++
			if i > j {
				break
			}
			res, err = rt.Concat(t, res, rt.StringValue(sep))
			if err != nil {
				break Switch
			}
			var v rt.Value
			v, err = rt.Index(t, tblVal, rt.IntValue(i))
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
	_, err := c.TableArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	tblVal := c.Arg(0)
	var (
		val rt.Value
		pos int64
	)
	tblLen, err := rt.IntLen(t, tblVal)
	if err != nil {
		return nil, err.AddContext(c)
	}
	if c.NArgs() >= 3 {
		pos, err = c.IntArg(1)
		if err != nil {
			return nil, err.AddContext(c)
		}
		if pos <= 0 || pos > tblLen+1 {
			return nil, rt.NewErrorS("#2 out of range").AddContext(c)
		}
		val = c.Arg(2)
	} else {
		pos = tblLen + 1
		val = c.Arg(1)
	}
	var (
		oldVal rt.Value
		posVal = rt.IntValue(pos)
	)
	for pos <= tblLen {
		oldVal, err = rt.Index(t, tblVal, posVal)
		if err != nil {
			return nil, err.AddContext(c)
		}
		err = rt.SetIndex(t, tblVal, posVal, val)
		if err != nil {
			return nil, err.AddContext(c)
		}
		val = oldVal
		pos++
		posVal = rt.IntValue(pos)
	}
	err = rt.SetIndex(t, tblVal, posVal, val)
	if err != nil {
		return nil, err.AddContext(c)
	}
	return c.Next(), nil
}

func move(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(4); err != nil {
		return nil, err.AddContext(c)
	}
	_, err := c.TableArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	srcVal := c.Arg(0)
	srcStart, err := c.IntArg(1)
	if err != nil {
		return nil, err.AddContext(c)
	}
	srcEnd, err := c.IntArg(2)
	if err != nil {
		return nil, err.AddContext(c)
	}
	dstStart, err := c.IntArg(3)
	if err != nil {
		return nil, err.AddContext(c)
	}
	dstVal := srcVal
	if c.NArgs() >= 5 {
		_, err = c.TableArg(4)
		if err != nil {
			return nil, err.AddContext(c)
		}
		dstVal = c.Arg(4)
	}
	if srcStart > srcEnd {
		// Nothing to do apparently!
	} else if dstStart >= srcStart {
		// Move in descending order to avoid writing at a position
		// before moving it
		dstStart += srcEnd - srcStart
		for srcEnd >= srcStart {
			v, err := rt.Index(t, srcVal, rt.IntValue(srcEnd))
			if err == nil {
				err = rt.SetIndex(t, dstVal, rt.IntValue(dstStart), v)
			}
			if err != nil {
				return nil, err.AddContext(c)
			}
			srcEnd--
			dstStart--
		}
	} else {
		// Move in ascending order to avoid writing at a position
		// before moving it
		for srcStart <= srcEnd {
			v, err := rt.Index(t, srcVal, rt.IntValue(srcStart))
			if err == nil {
				err = rt.SetIndex(t, dstVal, rt.IntValue(dstStart), v)
			}
			if err != nil {
				return nil, err.AddContext(c)
			}
			srcStart++
			dstStart++
		}
	}
	return c.PushingNext(dstVal), nil
}

func pack(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	tbl := rt.NewTable()
	// We can use tbl.Set() because tbl has no metatable
	for i, v := range c.Etc() {
		tbl.Set(rt.IntValue(int64(i+1)), v)
	}
	tbl.Set(rt.StringValue("n"), rt.IntValue(int64(len(c.Etc()))))
	return c.PushingNext(rt.TableValue(tbl)), nil
}

func remove(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	_, err := c.TableArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	tblVal := c.Arg(0)
	tblLen, err := rt.IntLen(t, tblVal)
	if err != nil {
		return nil, err.AddContext(c)
	}
	pos := tblLen
	if c.NArgs() >= 2 {
		pos, err = c.IntArg(1)
		if err != nil {
			return nil, err.AddContext(c)
		}
	}
	var val rt.Value
	switch {
	case pos == tblLen || pos == tblLen+1:
		posVal := rt.IntValue(pos)
		val, err = rt.Index(t, tblVal, posVal)
		if err == nil {
			err = rt.SetIndex(t, tblVal, posVal, rt.NilValue)
		}
		if err != nil {
			return nil, err.AddContext(c)
		}
	case pos <= 0 || pos > tblLen:
		return nil, rt.NewErrorS("#2 out of range").AddContext(c)
	default:
		var newVal rt.Value
		for pos <= tblLen {
			tblLenVal := rt.IntValue(tblLen)
			val, err = rt.Index(t, tblVal, tblLenVal)
			if err == nil {
				err = rt.SetIndex(t, tblVal, tblLenVal, newVal)
			}
			if err != nil {
				return nil, err.AddContext(c)
			}
			tblLen--
			newVal = val
		}
	}
	return c.PushingNext(val), nil
}

type tableSorter struct {
	len  func() int
	less func(i, j int) bool
	swap func(i, j int)
}

func (s *tableSorter) Less(i, j int) bool {
	return s.less(i, j)
}

func (s *tableSorter) Swap(i, j int) {
	s.swap(i, j)
}

func (s *tableSorter) Len() int {
	return s.len()
}

func sortf(t *rt.Thread, c *rt.GoCont) (next rt.Cont, resErr *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	_, err := c.TableArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	tblVal := c.Arg(0)
	get := func(i int) rt.Value {
		x, err := rt.Index(t, tblVal, rt.IntValue(int64(i+1)))
		if err != nil {
			panic(err)
		}
		return x
	}
	set := func(i int, x rt.Value) {
		err := rt.SetIndex(t, tblVal, rt.IntValue(int64(i+1)), x)
		if err != nil {
			panic(err)
		}
	}
	swap := func(i, j int) {
		x, y := get(i), get(j)
		set(i, y)
		set(j, x)
	}
	len := func() int {
		l, err := rt.IntLen(t, tblVal)
		if err != nil {
			panic(err)
		}
		return int(l)
	}
	var less func(i, j int) bool
	if c.NArgs() >= 2 {
		comp := c.Arg(1)
		term := rt.NewTerminationWith(1, false)
		less = func(i, j int) bool {
			term.Reset()
			err := rt.Call(t, comp, []rt.Value{get(i), get(j)}, term)
			if err != nil {
				panic(err)
			}
			return rt.Truth(term.Get(0))
		}
	} else {
		less = func(i, j int) bool {
			res, err := rt.Lt(t, get(i), get(j))
			if err != nil {
				panic(err)
			}
			return res
		}
	}
	defer func() {
		if err := recover(); err != nil {
			next = nil
			resErr = err.(*rt.Error).AddContext(c)
		}
	}()
	sorter := &tableSorter{len, less, swap}
	sort.Sort(sorter)
	return c.Next(), nil
}

func unpack(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	_, err := c.TableArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	tblVal := c.Arg(0)
	var (
		i int64 = 1
		j int64
	)
	nargs := c.NArgs()
	if nargs >= 2 {
		i, err = c.IntArg(1)
		if err != nil {
			return nil, err.AddContext(c)
		}
	}
	if nargs >= 3 && !c.Arg(2).IsNil() {
		j, err = c.IntArg(2)
	} else {
		j, err = rt.IntLen(t, tblVal)
	}
	if err != nil {
		return nil, err.AddContext(c)
	}
	next := c.Next()
	for ; i <= j; i++ {
		val, err := rt.Index(t, tblVal, rt.IntValue(i))
		if err != nil {
			return nil, err.AddContext(c)
		}
		next.Push(val)
	}
	return next, nil
}
