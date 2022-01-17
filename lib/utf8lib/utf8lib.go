package utf8lib

import (
	"errors"
	"unicode/utf8"

	"github.com/arnodel/golua/lib/packagelib"
	"github.com/arnodel/golua/luastrings"
	rt "github.com/arnodel/golua/runtime"
)

// LibLoader can load the utf8 lib.
var LibLoader = packagelib.Loader{
	Load: load,
	Name: "utf8",
}

func load(r *rt.Runtime) (rt.Value, func()) {
	pkg := rt.NewTable()
	r.SetEnv(pkg, "charpattern", rt.StringValue("[\x00-\x7F\xC2-\xF4][\x80-\xBF]*"))

	rt.SolemnlyDeclareCompliance(
		rt.ComplyCpuSafe|rt.ComplyMemSafe|rt.ComplyTimeSafe|rt.ComplyIoSafe,

		r.SetEnvGoFunc(pkg, "char", char, 0, true),
		r.SetEnvGoFunc(pkg, "codes", codes, 2, false),
		r.SetEnvGoFunc(pkg, "codepoint", codepoint, 4, false),
		r.SetEnvGoFunc(pkg, "len", lenf, 4, false),
		r.SetEnvGoFunc(pkg, "offset", offset, 3, false),
	)

	return rt.TableValue(pkg), nil
}

func char(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	runes := c.Etc()
	maxLen := len(runes) * utf8.UTFMax
	t.RequireBytes(maxLen)
	buf := make([]byte, maxLen)
	cur := buf
	bufLen := 0
	t.RequireCPU(uint64(len(runes)))
	for i, r := range runes {
		n, ok := rt.ToInt(r)
		if !ok {
			return nil, rt.NewErrorF("#%d should be an integer", i+1)
		}
		if n < 0 || n > 0x10FFFF {
			return nil, rt.NewErrorF("#%d value out of range", i+1)
		}
		sz := utf8.EncodeRune(cur, rune(n))
		cur = cur[sz:]
		bufLen += sz
	}
	t.ReleaseBytes(maxLen - bufLen)
	return c.PushingNext1(t.Runtime, rt.StringValue(string(buf[:bufLen]))), nil
}

func codes(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var (
		s   string
		lax bool
		err *rt.Error
	)
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	s, err = c.StringArg(0)
	if err != nil {
		return nil, err
	}
	if c.NArgs() >= 2 {
		lax, err = c.BoolArg(1)
		if err != nil {
			return nil, err
		}
	}
	decode := luastrings.GetDecodeRuneInString(lax)
	var p int64
	var iterF = func(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
		t.RequireCPU(1)
		next := c.Next()
		r, n := decode(s[p:])
		if r == utf8.RuneError {
			switch n {
			case 0:
				return next, nil
			case 1:
				return nil, rt.NewErrorE(errInvalidCode)
			}
			// If n > 1, then it is a successful decode in lax mode.
		}
		t.Push1(next, rt.IntValue(p+1))
		t.Push1(next, rt.IntValue(int64(r)))
		p += int64(n)
		return next, nil
	}
	var iter = rt.NewGoFunction(iterF, "codesiterator", 0, false)
	iter.SolemnlyDeclareCompliance(rt.ComplyCpuSafe | rt.ComplyMemSafe | rt.ComplyTimeSafe | rt.ComplyIoSafe)
	return c.PushingNext1(t.Runtime, rt.FunctionValue(iter)), nil
}

func codepoint(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	var ii int64 = 1
	s, err := c.StringArg(0)
	if err == nil && c.NArgs() >= 2 {
		ii, err = c.IntArg(1)
	}
	jj := ii
	if err == nil && c.NArgs() >= 3 {
		jj, err = c.IntArg(2)
	}
	lax := false
	if err == nil && c.NArgs() >= 4 {
		lax, err = c.BoolArg(3)
	}
	if err != nil {
		return nil, err
	}
	decode := luastrings.GetDecodeRuneInString(lax)
	next := c.Next()
	i := rt.StringNormPos(s, int(ii))
	if i < 1 {
		return nil, rt.NewErrorE(errPosOutOfRange)
	}
	j := rt.StringNormPos(s, int(jj))
	if j > len(s) {
		return nil, rt.NewErrorE(errPosOutOfRange)
	}
	for k := i - 1; k < j; {
		t.RequireCPU(1)
		r, sz := decode(s[k:])
		if r == utf8.RuneError && sz <= 1 {
			return nil, rt.NewErrorE(errInvalidCode)
		}
		t.Push1(next, rt.IntValue(int64(r)))
		k += sz
	}
	return next, nil
}

func lenf(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	var ii int64 = 1
	s, err := c.StringArg(0)
	if err == nil && c.NArgs() >= 2 {
		ii, err = c.IntArg(1)
	}
	var jj int64 = -1
	if err == nil && c.NArgs() >= 3 {
		jj, err = c.IntArg(2)
	}
	var lax = false
	if err == nil && c.NArgs() >= 4 {
		lax, err = c.BoolArg(3)
	}
	if err != nil {
		return nil, err
	}
	var (
		decode = luastrings.GetDecodeRuneInString(lax)
		next   = c.Next()
		i      = rt.StringNormPos(s, int(ii))
		j      = rt.StringNormPos(s, int(jj))
		slen   int64
	)
	for k := i - 1; k < j; {
		t.RequireCPU(1)
		r, sz := decode(s[k:])
		if r == utf8.RuneError && sz <= 1 {
			t.Push1(next, rt.NilValue)
			t.Push1(next, rt.IntValue(int64(k+1)))
			return next, nil
		}
		k += sz
		slen++
	}
	t.Push1(next, rt.IntValue(slen))
	return next, nil
}

func offset(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err
	}
	var nn int64
	ss, err := c.StringArg(0)
	if err == nil {
		nn, err = c.IntArg(1)
	}
	var ii int64 = 1
	if nn < 0 {
		ii = int64(len(ss) + 1)
	}
	if err == nil && c.NArgs() >= 3 {
		ii, err = c.IntArg(2)
	}
	if err != nil {
		return nil, err
	}
	i := rt.StringNormPos(ss, int(ii)) - 1
	s := string(ss)
	if i < 0 || i > len(s) {
		return nil, rt.NewErrorS("position out of range")
	}
	if nn == 0 {
		// Special case: locate the starting position of the current
		// code point.
		for i >= 0 && i < len(s) && !utf8.RuneStart(s[i]) {
			t.RequireCPU(1)
			i--
		}
	} else {
		if i < len(s) && !utf8.RuneStart(s[i]) {
			return nil, rt.NewErrorS("initial position is a continuation byte")
		}
		if nn > 0 {
			nn--
			// Go forward
			for nn > 0 {
				t.RequireCPU(1)
				i++
				if i >= len(s) {
					nn--
					break
				}
				if utf8.RuneStart(s[i]) {
					nn--
				}
			}
		} else {
			// Go backward
			for nn < 0 && i > 0 {
				t.RequireCPU(1)
				i--
				if utf8.RuneStart(s[i]) {
					nn++
				}
			}
		}
	}
	if nn == 0 {
		return c.PushingNext1(t.Runtime, rt.IntValue(int64(i+1))), nil
	}
	return c.PushingNext1(t.Runtime, rt.NilValue), nil
}

var errInvalidCode = errors.New("invalid UTF-8 code")
var errPosOutOfRange = errors.New("position out of range")
