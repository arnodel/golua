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
	clos, err := rt.CompileAndLoadLuaChunk(chunkName, chunk, rt.TableValue(chunkEnv))
	if err != nil {
		return nil, rt.NewErrorE(err).AddContext(c)
	}
	c.Next().Push(rt.FunctionValue(clos))
	return c.Next(), nil
}
