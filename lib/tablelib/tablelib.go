package tablelib

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/arnodel/golua/lib/packagelib"

	rt "github.com/arnodel/golua/runtime"
)

// LibLoader can load the table lib.
var LibLoader = packagelib.Loader{
	Load: load,
	Name: "table",
}

func load(r *rt.Runtime) (rt.Value, func()) {
	pkg := rt.NewTable()

	rt.SolemnlyDeclareCompliance(
		rt.ComplyCpuSafe|rt.ComplyMemSafe|rt.ComplyTimeSafe|rt.ComplyIoSafe,

		r.SetEnvGoFunc(pkg, "concat", concat, 4, false),
		r.SetEnvGoFunc(pkg, "insert", insert, 3, false),
		r.SetEnvGoFunc(pkg, "move", move, 5, false),
		r.SetEnvGoFunc(pkg, "pack", pack, 0, true),
		r.SetEnvGoFunc(pkg, "remove", remove, 2, false),
		r.SetEnvGoFunc(pkg, "sort", sortf, 2, false),
		r.SetEnvGoFunc(pkg, "unpack", unpack, 3, false),
	)

	return rt.TableValue(pkg), nil
}

func concat(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	_, err := c.TableArg(0)
	if err != nil {
		return nil, err
	}
	tblVal := c.Arg(0)
	var (
		sep string
		i   int64 = 1
	)
	j, err := rt.IntLen(t, tblVal)
	if err != nil {
		return nil, err
	}
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
		var item rt.Value
		if i > j {
			return c.PushingNext1(t.Runtime, rt.StringValue("")), nil
		}
		item, err = rt.Index(t, tblVal, rt.IntValue(i))
		if err != nil {
			break
		}
		var sb strings.Builder
		s, ok := item.ToString()
		if !ok {
			return nil, errInvalidConcatValue(item, i)
		}
		t.RequireBytes(len(s))
		sb.WriteString(s)
		for {
			// Don't require CPU because rt.Index will do
			if i == math.MaxInt64 {
				break
			}
			i++
			if i > j {
				break
			}
			t.RequireBytes(len(sep))
			sb.WriteString(sep)
			item, err = rt.Index(t, tblVal, rt.IntValue(i))
			if err != nil {
				return nil, err
			}
			s, ok = item.ToString()
			if !ok {
				return nil, errInvalidConcatValue(item, i)
			}
			t.RequireBytes(len(s))
			sb.WriteString(s)
		}
		return c.PushingNext1(t.Runtime, rt.StringValue(sb.String())), nil
	}
	return nil, err
}

func errInvalidConcatValue(v rt.Value, i int64) error {
	s, _ := v.ToString()
	return fmt.Errorf("invalid value (%s) at index %d in table for 'concat'", s, i)
}

func insert(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err
	}
	_, err := c.TableArg(0)
	if err != nil {
		return nil, err
	}
	tblVal := c.Arg(0)
	var (
		val rt.Value
		pos int64
	)
	tblLen, err := rt.IntLen(t, tblVal)
	if err != nil {
		return nil, err
	}
	if c.NArgs() >= 3 {
		pos, err = c.IntArg(1)
		if err != nil {
			return nil, err
		}
		if pos <= 0 || pos > tblLen+1 {
			return nil, errors.New("#2 out of range")
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
		// Don't require CPU because rt.Index and rt.SetIndex will do
		oldVal, err = rt.Index(t, tblVal, posVal)
		if err != nil {
			return nil, err
		}
		err = rt.SetIndex(t, tblVal, posVal, val)
		if err != nil {
			return nil, err
		}
		val = oldVal
		pos++
		posVal = rt.IntValue(pos)
	}
	err = rt.SetIndex(t, tblVal, posVal, val)
	if err != nil {
		return nil, err
	}
	return c.Next(), nil
}

func move(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.CheckNArgs(4); err != nil {
		return nil, err
	}
	_, err := c.TableArg(0)
	if err != nil {
		return nil, err
	}
	srcVal := c.Arg(0)
	srcStart, err := c.IntArg(1)
	if err != nil {
		return nil, err
	}
	srcEnd, err := c.IntArg(2)
	if err != nil {
		return nil, err
	}
	dstStart, err := c.IntArg(3)
	if err != nil {
		return nil, err
	}
	dstVal := srcVal
	if c.NArgs() >= 5 {
		_, err = c.TableArg(4)
		if err != nil {
			return nil, err
		}
		dstVal = c.Arg(4)
	}
	if srcStart > srcEnd || srcStart == dstStart && dstVal == srcVal {
		// Nothing to do apparently!
	} else if srcStart <= 0 && srcStart+math.MaxInt64 <= srcEnd {
		return nil, errors.New("interval too large")
	} else if dstStart >= srcStart {
		// Move in descending order to avoid writing at a position
		// before moving it
		offset := srcEnd - srcStart // 0 <= offset < math.MaxInt64
		if dstStart > math.MaxInt64-offset {
			// Not enough space to move
			return nil, errors.New("destination would wrap around")
		}
		dstStart += offset
		for srcEnd >= srcStart {
			// Don't require CPU because rt.Index and rt.SetIndex will do
			v, err := rt.Index(t, srcVal, rt.IntValue(srcEnd))
			if err == nil {
				err = rt.SetIndex(t, dstVal, rt.IntValue(dstStart), v)
			}
			if err != nil {
				return nil, err
			}
			if srcEnd == math.MinInt64 {
				// Prevent wrapping around
				break
			}
			srcEnd--
			dstStart--
		}
	} else {
		// Move in ascending order to avoid writing at a position
		// before moving it
		for srcStart <= srcEnd {
			// Don't require CPU because rt.Index and rt.SetIndex will do
			v, err := rt.Index(t, srcVal, rt.IntValue(srcStart))
			if err == nil {
				err = rt.SetIndex(t, dstVal, rt.IntValue(dstStart), v)
			}
			if err != nil {
				return nil, err
			}
			if srcStart == math.MaxInt64 {
				// Prevent wrapping around
				break
			}
			srcStart++
			dstStart++
		}
	}
	return c.PushingNext1(t.Runtime, dstVal), nil
}

func pack(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	tbl := rt.NewTable()
	// We can use t.SetTable() because tbl has no metatable
	for i, v := range c.Etc() {
		// SetTable always consumes CPU so the loop is protected.
		t.SetTable(tbl, rt.IntValue(int64(i+1)), v)
	}
	t.SetTable(tbl, rt.StringValue("n"), rt.IntValue(int64(len(c.Etc()))))
	return c.PushingNext1(t.Runtime, rt.TableValue(tbl)), nil
}

func remove(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	_, err := c.TableArg(0)
	if err != nil {
		return nil, err
	}
	tblVal := c.Arg(0)
	tblLen, err := rt.IntLen(t, tblVal)
	if err != nil {
		return nil, err
	}
	pos := tblLen
	if c.NArgs() >= 2 {
		pos, err = c.IntArg(1)
		if err != nil {
			return nil, err
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
			return nil, err
		}
	case pos <= 0 || pos > tblLen:
		return nil, errors.New("#2 out of range")
	default:
		var newVal rt.Value
		for pos <= tblLen {
			// Don't require CPU because rt.Index and rt.SetIndex will do
			tblLenVal := rt.IntValue(tblLen)
			val, err = rt.Index(t, tblVal, tblLenVal)
			if err == nil {
				err = rt.SetIndex(t, tblVal, tblLenVal, newVal)
			}
			if err != nil {
				return nil, err
			}
			tblLen--
			newVal = val
		}
	}
	return c.PushingNext1(t.Runtime, val), nil
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

const maxSortSize = 1 << 40

type sortError struct {
	err error
}

func throwSortError(err error) {
	panic(sortError{err: err})
}

func sortf(t *rt.Thread, c *rt.GoCont) (next rt.Cont, resErr error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	_, err := c.TableArg(0)
	if err != nil {
		return nil, err
	}
	tblVal := c.Arg(0)
	get := func(i int) rt.Value {
		x, err := rt.Index(t, tblVal, rt.IntValue(int64(i+1)))
		if err != nil {
			throwSortError(err)
		}
		return x
	}
	set := func(i int, x rt.Value) {
		err := rt.SetIndex(t, tblVal, rt.IntValue(int64(i+1)), x)
		if err != nil {
			throwSortError(err)
		}
	}
	swap := func(i, j int) {
		x, y := get(i), get(j)
		set(i, y)
		set(j, x)
	}
	l, err := rt.IntLen(t, tblVal)
	if err != nil {
		return nil, err
	}
	if l >= maxSortSize {
		return nil, errors.New("too big to sort")
	}
	if l <= 0 {
		return c.Next(), nil
	}
	len := func() int {
		return int(l)
	}
	var less func(i, j int) bool
	if c.NArgs() >= 2 && !c.Arg(1).IsNil() {
		comp := c.Arg(1)
		term := rt.NewTerminationWith(c, 1, false)
		less = func(i, j int) bool {
			term.Reset()
			err := rt.Call(t, comp, []rt.Value{get(i), get(j)}, term)
			if err != nil {
				throwSortError(err)
			}
			return rt.Truth(term.Get(0))
		}
	} else {
		less = func(i, j int) bool {
			res, err := rt.Lt(t, get(i), get(j))
			if err != nil {
				throwSortError(err)
			}
			return res
		}
	}
	defer func() {
		if r := recover(); r != nil {
			next = nil
			if sortErr, ok := r.(sortError); ok {
				resErr = sortErr.err
				return
			}
			panic(r)
		}
	}()
	sorter := &tableSorter{len, less, swap}
	// Because each operation on sorter consumes cpu resources, it's OK to call
	// sort.Sort.
	sort.Sort(sorter)
	return c.Next(), nil
}

// Maximum number of values that can be unpacked from a table.  Lua docs don't
// specify what this number should be.
const maxUnpackSize = 256

func unpack(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	_, err := c.TableArg(0)
	if err != nil {
		return nil, err
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
			return nil, err
		}
	}
	if nargs >= 3 && !c.Arg(2).IsNil() {
		j, err = c.IntArg(2)
	} else {
		j, err = rt.IntLen(t, tblVal)
	}
	if err != nil {
		return nil, err
	}
	if i < math.MaxInt64-maxUnpackSize && i+maxUnpackSize <= j {
		return nil, errors.New("too many values to unpack")
	}
	next := c.Next()
	for ; i <= j; i++ {
		// rt.Index consumes cpu so the loop is OK.
		val, err := rt.Index(t, tblVal, rt.IntValue(i))
		if err != nil {
			return nil, err
		}
		t.Push1(next, val)
		if i == math.MaxInt64 {
			// Prevent wrap around
			break
		}
	}
	return next, nil
}
