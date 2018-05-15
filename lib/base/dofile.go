package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func dofile(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	chunk, chunkName, err := loadChunk(c.Args())
	if err != nil {
		return nil, rt.NewErrorE(err).AddContext(c)
	}
	clos, err := rt.CompileLuaChunk(chunkName, chunk, t.GlobalEnv())
	if err != nil {
		return nil, rt.NewErrorE(err).AddContext(c)
	}
	return rt.ContWithArgs(clos, nil, c.Next()), nil
}
