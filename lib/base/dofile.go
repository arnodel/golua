package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func dofile(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	chunk, chunkName, err := loadChunk(t, c.Args())
	defer t.ReleaseBytes(len(chunk))
	if err != nil {
		return nil, rt.NewErrorE(err)
	}
	clos, err := t.CompileAndLoadLuaChunk(chunkName, chunk, t.GlobalEnv())
	if err != nil {
		return nil, rt.NewErrorE(err)
	}
	return t.ContWithArgs(clos, nil, c.Next()), nil
}
