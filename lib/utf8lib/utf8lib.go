package utf8lib

import (
	"errors"
	"unicode/utf8"

	"github.com/arnodel/golua/lib/packagelib"
	rt "github.com/arnodel/golua/runtime"
)

// LibLoader can load the utf8 lib.
var LibLoader = packagelib.Loader{
	Load: load,
	Name: "utf8",
}

func load(r *rt.Runtime) rt.Value {
	pkg := rt.NewTable()
	rt.SetEnvGoFunc(pkg, "char", char, 0, true)
	rt.SetEnv(pkg, "charpattern", rt.String("[\x00-\x7F\xC2-\xF4][\x80-\xBF]*"))
	rt.SetEnvGoFunc(pkg, "codes", codes, 1, false)
	rt.SetEnvGoFunc(pkg, "codepoint", codepoint, 3, false)
	rt.SetEnvGoFunc(pkg, "len", lenf, 3, false)
	rt.SetEnvGoFunc(pkg, "offset", offset, 3, false)
	return pkg
}

func char(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	runes := c.Etc()
	buf := make([]byte, len(runes)*utf8.UTFMax)
	cur := buf
	bufLen := 0
	for i, r := range runes {
		n, tp := rt.ToInt(r)
		if tp != rt.IsInt {
			return nil, rt.NewErrorF("#%d should be an integer", i+1).AddContext(c)
		}
		if n < 0 || n > 0x10FFFF {
			return nil, rt.NewErrorF("#%d value out of range", i+1).AddContext(c)
		}
		sz := utf8.EncodeRune(cur, rune(n))
		cur = cur[sz:]
		bufLen += sz
	}
	return c.PushingNext(rt.String(buf[:bufLen])), nil
}

func codes(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	ss, err := c.StringArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	s := []byte(ss)
	p := 0
	var iterF = func(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
		next := c.Next()
		if p < len(s) {
			r, n := utf8.DecodeRune(s[p:])
			if r == utf8.RuneError {
				switch n {
				case 0:
					return next, nil
				case 1:
					return nil, rt.NewErrorE(errInvalidCode).AddContext(c)
				}
			}
			next.Push(rt.Int(p + 1))
			next.Push(rt.Int(r))
			p += n
		}
		return next, nil
	}
	var iter = rt.NewGoFunction(iterF, "codesiterator", 0, false)
	return c.PushingNext(iter), nil
}

func codepoint(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	ii := rt.Int(1)
	ss, err := c.StringArg(0)
	if err == nil && c.NArgs() >= 2 {
		ii, err = c.IntArg(1)
	}
	jj := ii
	if err == nil && c.NArgs() >= 3 {
		jj, err = c.IntArg(2)
	}
	if err != nil {
		return nil, err.AddContext(c)
	}
	next := c.Next()
	i := ss.NormPos(ii)
	if i < 1 {
		return nil, rt.NewErrorE(errPosOutOfRange).AddContext(c)
	}
	j := ss.NormPos(jj)
	if j > len(ss) {
		return nil, rt.NewErrorE(errPosOutOfRange).AddContext(c)
	}
	s := string(ss)
	for k := i - 1; k < j; {
		r, sz := utf8.DecodeRuneInString(s[k:])
		if r == utf8.RuneError {
			return nil, rt.NewErrorE(errInvalidCode).AddContext(c)
		}
		next.Push(rt.Int(r))
		k += sz
	}
	return next, nil
}

func lenf(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	ii := rt.Int(1)
	ss, err := c.StringArg(0)
	if err == nil && c.NArgs() >= 2 {
		ii, err = c.IntArg(1)
	}
	jj := rt.Int(-1)
	if err == nil && c.NArgs() >= 3 {
		jj, err = c.IntArg(2)
	}
	if err != nil {
		return nil, err.AddContext(c)
	}
	next := c.Next()
	i := ss.NormPos(ii)
	j := ss.NormPos(jj)
	s := string(ss)
	slen := 0
	for k := i - 1; k < j; {
		r, sz := utf8.DecodeRuneInString(s[k:])
		if r == utf8.RuneError {
			next.Push(nil)
			next.Push(rt.Int(k + 1))
			return next, nil
		}
		k += sz
		slen++
	}
	next.Push(rt.Int(slen))
	return next, nil
}

func offset(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err.AddContext(c)
	}
	var nn rt.Int
	ss, err := c.StringArg(0)
	if err == nil {
		nn, err = c.IntArg(1)
	}
	ii := rt.Int(1)
	if nn < 0 {
		ii = rt.Int(len(ss) + 1)
	}
	if err == nil && c.NArgs() >= 3 {
		ii, err = c.IntArg(2)
	}
	if err != nil {
		return nil, err.AddContext(c)
	}
	i := ss.NormPos(ii) - 1
	s := string(ss)
	if i < 0 || i > len(s) {
		return nil, rt.NewErrorS("position out of range").AddContext(c)
	}
	if nn == 0 {
		// Special case: locate the starting position of the current
		// code point.
		for i >= 0 && i < len(s) && !utf8.RuneStart(s[i]) {
			i--
		}
	} else {
		if i < len(s) && !utf8.RuneStart(s[i]) {
			return nil, rt.NewErrorS("initial position is a continuation byte").AddContext(c)
		}
		if nn > 0 {
			nn--
			// Go forward
			for nn > 0 {
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
				i--
				if utf8.RuneStart(s[i]) {
					nn++
				}
			}
		}
	}
	if nn == 0 {
		return c.PushingNext(rt.Int(i + 1)), nil
	}
	return c.PushingNext(nil), nil
}

var errInvalidCode = errors.New("invalid UTF-8 code")
var errPosOutOfRange = errors.New("position out of range")
