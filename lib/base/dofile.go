package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func dofile(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := t.CheckIO(); err != nil {
		return nil, err.AddContext(c)
	}
	chunk, chunkName, err := loadChunk(t, c.Args())
	defer t.ReleaseBytes(len(chunk))
	if err != nil {
		return nil, rt.NewErrorE(err).AddContext(c)
	}
	clos, err := t.CompileAndLoadLuaChunk(chunkName, chunk, t.GlobalEnv())
	if err != nil {
		return nil, rt.NewErrorE(err).AddContext(c)
	}
	return t.ContWithArgs(clos, nil, c.Next()), nil
}
