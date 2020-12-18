package stringlib

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/arnodel/golua/lib/stringlib/pattern"
	rt "github.com/arnodel/golua/runtime"
)

func find(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var (
		s, ptn string
		init   int64 = 1
		plain  bool
	)
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
	si := rt.StringNormPos(s, int(init)) - 1
	next := c.Next()
	switch {
	case si < 0 || si > len(s):
		next.Push(rt.NilValue)
	case plain || len(ptn) == 0:
		i := strings.Index(string(s)[si:], string(ptn))
		if i == -1 {
			next.Push(rt.NilValue)
		} else {
			next.Push(rt.IntValue(int64(i + 1)))
			next.Push(rt.IntValue(int64(i + len(ptn))))
		}
	default:
		pat, err := pattern.New(string(ptn))
		if err != nil {
			return nil, rt.NewErrorE(err).AddContext(c)
		}
		captures := pat.MatchFromStart(string(s), si)
		if len(captures) == 0 {
			next.Push(rt.NilValue)
		} else {
			first := captures[0]
			next.Push(rt.IntValue(int64(first.Start() + 1)))
			next.Push(rt.IntValue(int64(first.End())))
			pushExtraCaptures(captures, s, next)
		}
	}
	return next, nil
}

func match(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var (
		s, ptn string
		init   int64 = 1
	)
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
	si := rt.StringNormPos(s, int(init)) - 1
	next := c.Next()
	pat, ptnErr := pattern.New(string(ptn))
	if ptnErr != nil {
		return nil, rt.NewErrorE(ptnErr).AddContext(c)
	}
	pushCaptures(pat.MatchFromStart(string(s), si), s, next)
	return next, nil
}

func pushCaptures(captures []pattern.Capture, s string, next rt.Cont) {
	if len(captures) == 0 {
		next.Push(rt.NilValue)
	} else if len(captures) == 1 {
		c := captures[0]
		next.Push(rt.StringValue(s[c.Start():c.End()]))
	} else {
		pushExtraCaptures(captures, s, next)
	}
}

func pushExtraCaptures(captures []pattern.Capture, s string, next rt.Cont) {
	if len(captures) < 2 {
		return
	}
	for _, c := range captures[1:] {
		next.Push(captureValue(c, s))
	}
}

func captureValue(c pattern.Capture, s string) rt.Value {
	if c.IsEmpty() {
		return rt.IntValue(int64(c.Start() + 1))
	}
	return rt.StringValue(s[c.Start():c.End()])
}

func gmatch(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var s, ptn string
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
	return c.PushingNext(rt.FunctionValue(rt.NewGoFunction(iterator, "gmatchiterator", 0, false))), nil
}

func gsub(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var (
		s, ptn string
		n      int64 = -1
		repl   rt.Value
	)
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

	if replString, ok := repl.TryString(); ok {
		replF = func(captures []pattern.Capture) (string, *rt.Error) {
			cStrings := [10]string{}
			maxIndex := len(captures) - 1
			for i, c := range captures {
				v := captureValue(c, s)
				switch v.Type() {
				case rt.StringType:
					cStrings[i] = v.AsString()
				case rt.IntType:
					cStrings[i] = strconv.Itoa(int(v.AsInt()))
				}
			}
			if len(captures) == 1 {
				cStrings[1] = cStrings[0]
				maxIndex = 1
			}
			var err *rt.Error
			return gsubPtn.ReplaceAllStringFunc(string(replString), func(x string) string {
				if err != nil {
					return ""
				}
				b := x[1]
				switch {
				case '0' <= b && b <= '9':
					idx := int(b - '0')
					if idx > maxIndex {
						err = rt.NewErrorE(pattern.ErrInvalidCaptureIdx(idx)).AddContext(c)
						return ""
					}
					return cStrings[b-'0']
				case b == '%':
					return x[1:]
				default:
					err = rt.NewErrorE(pattern.ErrInvalidPct).AddContext(c)
				}
				return x[1:]
			}), err
		}
	} else if replTable, ok := repl.TryTable(); ok {
		replF = func(captures []pattern.Capture) (string, *rt.Error) {
			gc := captures[0]
			i := 0
			if len(captures) >= 2 {
				i = 1
			}
			c := captures[i]
			val, err := rt.Index(t, rt.TableValue(replTable), captureValue(c, s))
			if err != nil {
				return "", err
			}
			return subToString(s[gc.Start():gc.End()], val)
		}
	} else if replC, ok := repl.TryCallable(); ok {
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
	var (
		si         int
		sb         strings.Builder
		matchCount int64
		allowEmpty = true
	)
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
	next.Push(rt.StringValue(sb.String()))
	next.Push(rt.IntValue(matchCount))
	return next, nil
}

var gsubPtn = regexp.MustCompile("%.")

func subToString(key string, val rt.Value) (string, *rt.Error) {
	if !rt.Truth(val) {
		return key, nil
	}
	res, ok := rt.ToString(val)
	if ok {
		return string(res), nil
	}
	return "", rt.NewErrorF("invalid replacement value (a %s)", rt.Type(val))
}
