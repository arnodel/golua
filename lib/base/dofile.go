package base

import (
	"fmt"

	rt "github.com/arnodel/golua/runtime"
)

func dofile(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	chunk, chunkName, err := loadChunk(c.Args())
	if err != nil {
		return nil, fmt.Errorf("dofile: %s", err)
	}
	// TODO: use chunkName
	_ = chunkName
	clos, err := rt.CompileLuaChunk(chunk, t.GlobalEnv())
	if err != nil {
		return c, err
	}
	return rt.ContWithArgs(clos, nil, c.Next()), nil
}
