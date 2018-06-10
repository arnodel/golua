package stringlib

import (
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
		captures := pat.Match(string(s), si)
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
	captures := pat.Match(string(s), si)
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
	return next, nil
}
