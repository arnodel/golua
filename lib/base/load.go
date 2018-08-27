package base

import (
	"bytes"
	"strings"

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
	// TODO: use chunkMode
	canBeBinary := strings.IndexByte(chunkMode, 'b') >= 0
	canBeText := strings.IndexByte(chunkMode, 't') >= 0
	if len(chunk) > 0 && chunk[0] < rt.ConstTypeMaj {
		// binary chunk
		if !canBeBinary {
			return nil, rt.NewErrorF("Did not expect binary chunk").AddContext(c)
		}
		r := bytes.NewBuffer(chunk)
		k, err := rt.LoadConst(r)
		if err != nil {
			return nil, rt.NewErrorE(err).AddContext(c)
		}
		code, ok := k.(*rt.Code)
		if !ok {
			return nil, rt.NewErrorF("Expected function to load").AddContext(c)
		}
		clos := rt.NewClosure(code)
		if code.UpvalueCount > 0 {
			var envVal rt.Value = chunkEnv
			clos.AddUpvalue(rt.NewCell(envVal))
		}
		return c.PushingNext(clos), nil
	} else if !canBeText {
		return nil, rt.NewErrorF("Did not expect text chunk").AddContext(c)
	}
	clos, err := rt.CompileAndLoadLuaChunk(chunkName, chunk, chunkEnv)
	next := c.Next()
	if err != nil {
		next.Push(nil)
		next.Push(rt.String(err.Error()))
	} else {
		next.Push(clos)
	}
	return next, nil
}
