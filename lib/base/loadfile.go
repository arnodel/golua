package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func loadfile(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := t.CheckIO(); err != nil {
		return nil, err.AddContext(c)
	}
	chunk, chunkName, err := loadChunk(t, c.Args())
	if err != nil {
		return nil, rt.NewErrorE(err).AddContext(c)
	}
	var chunkMode string
	var chunkEnv = t.GlobalEnv()
	switch nargs := c.NArgs(); {
	case nargs >= 3:
		var err *rt.Error
		chunkEnv, err = c.TableArg(2)
		if err != nil {
			return nil, err.AddContext(c)
		}
		fallthrough
	case nargs >= 2:
		mode, err := c.StringArg(1)
		if err != nil {
			return nil, err.AddContext(c)
		}
		chunkMode = string(mode)
	}
	// TODO: use mode
	_ = chunkMode
	clos, err := t.CompileAndLoadLuaChunk(chunkName, chunk, chunkEnv)
	if err != nil {
		return nil, rt.NewErrorE(err).AddContext(c)
	}
	return c.PushingNext1(t.Runtime, rt.FunctionValue(clos)), nil
}
