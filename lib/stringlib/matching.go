package stringlib

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/arnodel/golua/lib/stringlib/pattern"
	rt "github.com/arnodel/golua/runtime"
)

func find(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var s, ptn rt.String
	var init rt.Int = 1
	var plain bool

	err := c.CheckNArgs(2)
	if err == nil {
		s, err = c.StringArg(0)
	}
	if err == nil {
		ptn, err = c.StringArg(1)
	}
	if err == nil && c.NArgs() >= 3 {
		init, err = c.IntArg(2)
		if err == nil && c.NArgs() >= 4 {
			plain = rt.Truth(c.Arg(3))
		}
	}
	if err != nil {
		return nil, err.AddContext(c)
	}
	si := pos(s, init) - 1
	next := c.Next()
	switch {
	case si < 0 || si > len(s):
		next.Push(nil)
	case plain || len(ptn) == 0:
		i := strings.Index(string(s)[si:], string(ptn))
		if i == -1 {
			next.Push(nil)
		} else {
			next.Push(rt.Int(i + 1))
			next.Push(rt.Int(i + len(ptn)))
		}
	default:
		pat, err := pattern.New(string(ptn))
		if err != nil {
			return nil, rt.NewErrorE(err).AddContext(c)
		}
		captures := pat.MatchFromStart(string(s), si)
		if len(captures) == 0 {
			next.Push(nil)
		} else {
			first := captures[0]
			next.Push(rt.Int(first.Start() + 1))
			next.Push(rt.Int(first.End()))
			pushExtraCaptures(captures, s, next)
		}
	}
	return next, nil
}

func match(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var s, ptn rt.String
	var init rt.Int = 1
	err := c.CheckNArgs(2)
	if err == nil {
		s, err = c.StringArg(0)
	}
	if err == nil {
		ptn, err = c.StringArg(1)
	}
	if err == nil && c.NArgs() >= 3 {
		init, err = c.IntArg(2)
	}
	if err != nil {
		return nil, err.AddContext(c)
	}
	si := pos(s, init) - 1
	next := c.Next()
	pat, ptnErr := pattern.New(string(ptn))
	if ptnErr != nil {
		return nil, rt.NewErrorE(ptnErr).AddContext(c)
	}
	pushCaptures(pat.MatchFromStart(string(s), si), s, next)
	return next, nil
}

func pushCaptures(captures []pattern.Capture, s rt.String, next rt.Cont) {
	if len(captures) == 0 {
		next.Push(nil)
	} else if len(captures) == 1 {
		c := captures[0]
		next.Push(s[c.Start():c.End()])
	} else {
		pushExtraCaptures(captures, s, next)
	}
}

func pushExtraCaptures(captures []pattern.Capture, s rt.String, next rt.Cont) {
	if len(captures) < 2 {
		return
	}
	for _, c := range captures[1:] {
		next.Push(captureValue(c, s))
	}
}

func captureValue(c pattern.Capture, s rt.String) rt.Value {
	if c.IsEmpty() {
		return rt.Int(c.Start() + 1)
	}
	return s[c.Start():c.End()]
}

func gmatch(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var s, ptn rt.String
	err := c.CheckNArgs(2)
	if err == nil {
		s, err = c.StringArg(0)
	}
	if err == nil {
		ptn, err = c.StringArg(1)
	}
	if err != nil {
		return nil, err.AddContext(c)
	}
	pat, ptnErr := pattern.New(string(ptn))
	if ptnErr != nil {
		return nil, rt.NewErrorE(ptnErr).AddContext(c)
	}
	si := 0
	allowEmpty := true
	var iterator = func(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
		next := c.Next()
		var captures []pattern.Capture
		for {
			captures = pat.Match(string(s), si)
			if len(captures) == 0 {
				break
			}
			gc := captures[0]
			start, end := gc.Start(), gc.End()
			if allowEmpty || start != si || end != si {
				allowEmpty = start >= end
				if allowEmpty {
					si = start + 1
				} else {
					si = end
				}
				break
			}
			si++
			allowEmpty = true
		}
		pushCaptures(captures, s, next)
		return next, nil
	}
	return c.PushingNext(rt.NewGoFunction(iterator, "gmatchiterator", 0, false)), nil
}

func gsub(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var s, ptn rt.String
	var n rt.Int = -1
	var repl rt.Value
	err := c.CheckNArgs(3)
	if err == nil {
		s, err = c.StringArg(0)
	}
	if err == nil {
		ptn, err = c.StringArg(1)
	}
	if err == nil && c.NArgs() >= 4 {
		n, err = c.IntArg(3)
	}
	if err != nil {
		return nil, err.AddContext(c)
	}
	repl = c.Arg(2)
	pat, ptnErr := pattern.New(string(ptn))
	if ptnErr != nil {
		return nil, rt.NewErrorE(ptnErr).AddContext(c)
	}
	var replF func([]pattern.Capture) (string, *rt.Error)

	if replString, ok := repl.(rt.String); ok {
		replF = func(captures []pattern.Capture) (string, *rt.Error) {
			cStrings := [10]string{}
			for i, c := range captures {
				v := captureValue(c, s)
				switch vv := v.(type) {
				case rt.String:
					cStrings[i] = string(vv)
				case rt.Int:
					cStrings[i] = strconv.Itoa(int(vv))
				}
			}
			if len(captures) == 1 {
				cStrings[1] = cStrings[0]
			}
			return gsubPtn.ReplaceAllStringFunc(string(replString), func(x string) string {
				b := x[1]
				if '0' <= b && b <= '9' {
					return cStrings[b-'0']
				}
				return x[1:]
			}), nil
		}
	} else if replTable, ok := repl.(*rt.Table); ok {
		replF = func(captures []pattern.Capture) (string, *rt.Error) {
			gc := captures[0]
			i := 0
			if len(captures) >= 2 {
				i = 1
			}
			c := captures[i]
			val, err := rt.Index(t, replTable, captureValue(c, s))
			if err != nil {
				return "", err
			}
			return subToString(s[gc.Start():gc.End()], val)

		}
	} else if replC, ok := repl.(rt.Callable); ok {
		replF = func(captures []pattern.Capture) (string, *rt.Error) {
			term := rt.NewTerminationWith(1, false)
			cont := replC.Continuation(term)
			gc := captures[0]
			i := 0
			if len(captures) >= 2 {
				i = 1
			}
			for _, c := range captures[i:] {
				cont.Push(captureValue(c, s))
			}
			err := t.RunContinuation(cont)
			if err != nil {
				return "", err
			}
			return subToString(s[gc.Start():gc.End()], term.Get(0))
		}
	} else {
		return nil, rt.NewErrorS("#3 must be a string, table or function").AddContext(c)
	}
	si := 0
	var sb strings.Builder
	var matchCount rt.Int
	allowEmpty := true
	for ; matchCount != n; matchCount++ {
		captures := pat.Match(string(s), si)
		if len(captures) == 0 {
			break
		}
		gc := captures[0]
		start, end := gc.Start(), gc.End()
		if allowEmpty || start != si || end != si {
			sub, err := replF(captures)
			if err != nil {
				return nil, err.AddContext(c)
			}
			_, _ = sb.WriteString(string(s)[si:start])
			_, _ = sb.WriteString(sub)
		}
		allowEmpty = start >= end
		if allowEmpty {
			if start < len(s) {
				_ = sb.WriteByte(s[start])
			}
			si = start + 1
		} else {
			si = end
		}
	}
	if si < len(s) {
		_, _ = sb.WriteString(string(s)[si:])
	}
	next := c.Next()
	next.Push(rt.String(sb.String()))
	next.Push(matchCount)
	return next, nil
}

var gsubPtn = regexp.MustCompile("%.")

func subToString(key rt.String, val rt.Value) (string, *rt.Error) {
	if !rt.Truth(val) {
		return string(key), nil
	}
	res, ok := rt.AsString(val)
	if ok {
		return string(res), nil
	}
	return "", rt.NewErrorF("invalid replacement value (%s)", rt.Type(res))
}
