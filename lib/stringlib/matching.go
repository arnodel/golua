package stringlib

import (
	"regexp"
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
	if plain {
		i := strings.Index(string(s)[si:], string(ptn))
		if i == -1 {
			next.Push(nil)
		} else {
			next.Push(rt.Int(i + 1))
			next.Push(rt.Int(i + len(ptn)))
		}
	} else {
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
			for _, c := range captures[1:] {
				next.Push(s[c.Start():c.End()])
			}
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
		for _, c := range captures[1:] {
			next.Push(s[c.Start():c.End()])
		}
	}
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
	var iterator = func(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
		next := c.Next()
		captures := pat.Match(string(s), si)
		if len(captures) > 0 {
			si = captures[0].End()
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
				cStrings[i] = string(s)[c.Start():c.End()]
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
			key := s[c.Start():c.End()]
			val, err := rt.Index(t, replTable, key)
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
				cont.Push(s[c.Start():c.End()])
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
	for ; matchCount != n; matchCount++ {
		captures := pat.Match(string(s), si)
		if len(captures) == 0 {
			break
		}
		sub, err := replF(captures)
		if err != nil {
			return nil, err.AddContext(c)
		}
		gc := captures[0]
		_, _ = sb.WriteString(string(s)[si:gc.Start()])
		_, _ = sb.WriteString(sub)
		si = gc.End()
	}
	_, _ = sb.WriteString(string(s)[si:])
	next := c.Next()
	next.Push(rt.String(sb.String()))
	next.Push(matchCount)
	return next, nil
}

var gsubPtn = regexp.MustCompile("%.")

func subToString(key rt.String, val rt.Value) (string, *rt.Error) {
	res, ok := rt.AsString(val)
	if ok {
		return string(res), nil
	}
	if !rt.Truth(val) {
		return string(key), nil
	}
	return "", rt.NewErrorF("invalid replacement value (%s)", rt.Type(res))
}
