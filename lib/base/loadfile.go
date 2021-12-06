package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func loadfile(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	chunk, chunkName, err := loadChunk(t, c.Args())
	defer t.ReleaseBytes(len(chunk))
	if err != nil {
		return nil, rt.NewErrorE(err)
	}
	var chunkMode string
	var chunkEnv = rt.TableValue(t.GlobalEnv())
	switch nargs := c.NArgs(); {
	case nargs >= 3:
		chunkEnv = c.Arg(2)
		fallthrough
	case nargs >= 2:
		mode, err := c.StringArg(1)
		if err != nil {
			return nil, err
		}
		chunkMode = string(mode)
	}
	// TODO: use mode
	_ = chunkMode
	clos, err := t.CompileAndLoadLuaChunk(chunkName, chunk, chunkEnv)
	if err != nil {
		return nil, rt.NewErrorE(err)
	}
	return c.PushingNext1(t.Runtime, rt.FunctionValue(clos)), nil
}
