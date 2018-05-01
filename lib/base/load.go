package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func load(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var chunk []byte
	chunkName := "chunk"
	chunkMode := "bt"
	chunkEnv := t.GlobalEnv()

	switch nargs := c.NArgs(); {
	case nargs >= 4:
		env, ok := c.Arg(3).(*rt.Table)
		if !ok {
			return nil, rt.NewErrorS("#4 (env) must be a table").AddContext(c)
		}
		chunkEnv = env
		fallthrough
	case nargs >= 3:
		mode, ok := c.Arg(2).(rt.String)
		if !ok {
			return nil, rt.NewErrorS("#3 (mode) must be a string").AddContext(c)
		}
		chunkMode = string(mode)
		fallthrough
	case nargs >= 2:
		name, ok := c.Arg(1).(rt.String)
		if !ok {
			return nil, rt.NewErrorS("#2 (name) must be a string").AddContext(c)
		}
		chunkName = string(name)
		fallthrough
	case nargs >= 1:
		switch x := c.Arg(0).(type) {
		case rt.String:
			chunk = []byte(x)
		default:
			return nil, rt.NewErrorS("#1 (chunk) must be a string").AddContext(c)
		}
	}
	// TODO: use those
	_, _ = chunkName, chunkMode
	clos, err := rt.CompileLuaChunk(chunk, chunkEnv)
	if err != nil {
		return nil, rt.NewErrorE(err).AddContext(c)
	}
	c.Next().Push(clos)
	return c.Next(), nil
}
