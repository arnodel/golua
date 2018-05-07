package base

import (
	rt "github.com/arnodel/golua/runtime"
)

func load(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	var chunk []byte
	chunkName := "chunk"
	chunkMode := "bt"
	chunkEnv := t.GlobalEnv()

	switch nargs := c.NArgs(); {
	case nargs >= 4:
		var err *rt.Error
		chunkEnv, err = c.TableArg(3)
		if err != nil {
			return nil, err.AddContext(c)
		}
		fallthrough
	case nargs >= 3:
		mode, err := c.StringArg(2)
		if err != nil {
			return nil, err.AddContext(c)
		}
		chunkMode = string(mode)
		fallthrough
	case nargs >= 2:
		name, err := c.StringArg(1)
		if err != nil {
			return nil, err.AddContext(c)
		}
		chunkName = string(name)
		fallthrough
	case nargs >= 1:
		switch x := c.Arg(0).(type) {
		case rt.String:
			chunk = []byte(x)
			// TODO: function case
		default:
			return nil, rt.NewErrorS("#1 must be a string or function").AddContext(c)
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
