package utf8lib

import (
	"unicode/utf8"

	rt "github.com/arnodel/golua/runtime"
)

func Load(r *rt.Runtime) {
	pkg := rt.NewTable()
	rt.SetEnv(r.GlobalEnv(), "utf8", pkg)

	rt.SetEnvGoFunc(pkg, "char", char, 0, true)
	rt.SetEnv(pkg, "charpattern", `[\0-\x7F\xC2-\xF4][\x80-\xBF]*`)
	rt.SetEnvGoFunc(pkg, "codes", codes, 1, false)
	rt.SetEnvGoFunc(pkg, "codepoint", codepoint, 3, false)
}

func char(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	runes := c.Etc()
	buf := make([]byte, len(runes)*utf8.UTFMax)
	cur := buf
	bufLen := 0
	for i, r := range runes {
		n, tp := rt.ToInt(r)
		if tp != rt.IsInt {
			return nil, rt.NewErrorF("#%d should be an integer", i+1)
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
					return nil, rt.NewErrorS("Invalid utf8 encoding").AddContext(c)
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
	j := ss.NormPos(jj)
	s := string(ss)
	for k := i - 1; k < j; {
		r, sz := utf8.DecodeRuneInString(s[k:])
		if r == utf8.RuneError {
			return nil, rt.NewErrorS("Invalid utf8 encoding").AddContext(c)
		}
		next.Push(rt.Int(r))
		k += sz
	}
	return next, nil
}
