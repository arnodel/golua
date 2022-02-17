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
	clos, err := t.LoadFromSourceOrCode(chunkName, chunk, "bt", rt.TableValue(t.GlobalEnv()), true)
	if err != nil {
		return nil, rt.NewErrorE(err)
	}
	return clos.Continuation(t, c.Next()), nil
}
