package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func loadfile(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	chunk, chunkName, err := loadChunk(c.Args())
	if err != nil {
		return nil, rt.NewErrorE(err).AddContext(c)
	}
	var chunkMode string
	var chunkEnv = t.GlobalEnv()
	switch nargs := c.NArgs(); {
	case nargs >= 3:
		env, ok := c.Arg(2).(*rt.Table)
		if !ok {
			return nil, rt.NewErrorS("#3 (env) must be a table").AddContext(c)
		}
		chunkEnv = env
		fallthrough
	case nargs >= 2:
		mode, ok := c.Arg(1).(rt.String)
		if !ok {
			return nil, rt.NewErrorS("#2 (mode) must be a string").AddContext(c)
		}
		chunkMode = string(mode)
	}
	// TODO: use name and mode
	_, _ = chunkName, chunkMode
	clos, err := rt.CompileLuaChunk(chunk, chunkEnv)
	if err != nil {
		return nil, rt.NewErrorE(err).AddContext(c)
	}
	c.Next().Push(clos)
	return c.Next(), nil
}
