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
	var chunkEnv rt.Value = t.GlobalEnv()
	next := c.Next()

	switch nargs := c.NArgs(); {
	case nargs >= 4:
		chunkEnv = c.Arg(3)
		fallthrough
	case nargs >= 3:
		if c.Arg(2) != nil {
			mode, err := c.StringArg(2)
			if err != nil {
				return nil, err.AddContext(c)
			}
			chunkMode = string(mode)
		}
		fallthrough
	case nargs >= 2:
		if c.Arg(1) != nil {
			name, err := c.StringArg(1)
			if err != nil {
				return nil, err.AddContext(c)
			}
			chunkName = string(name)
		}
		fallthrough
	case nargs >= 1:
		switch x := c.Arg(0).(type) {
		case rt.String:
			chunk = []byte(x)
		case rt.Callable:
			var buf bytes.Buffer
			for {
				bit, err := rt.Call1(t, x)
				if err != nil {
					next.Push(nil)
					next.Push(rt.String(err.Error()))
					return next, nil
				}
				if bit == nil {
					break
				}
				bitString, ok := bit.(rt.String)
				if !ok {
					rt.Push(next, nil, rt.String("reader must return a string"))
					return next, nil
				}
				if len(bitString) == 0 {
					break
				}
				buf.WriteString(string(bitString))
			}
			chunk = buf.Bytes()
		default:
			return nil, rt.NewErrorS("#1 must be a string or function").AddContext(c)
		}
	}
	canBeBinary := strings.IndexByte(chunkMode, 'b') >= 0
	canBeText := strings.IndexByte(chunkMode, 't') >= 0
	if len(chunk) > 0 && chunk[0] < rt.ConstTypeMaj {
		// binary chunk
		if !canBeBinary {
			rt.Push(next, nil, rt.String("attempt to load a binary chunk"))
			return next, nil
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
		rt.Push(next, nil, rt.String("attempt to load a text chunk"))
		return next, nil
	}
	clos, err := rt.CompileAndLoadLuaChunk(chunkName, chunk, chunkEnv)
	if err != nil {
		rt.Push(next, nil, rt.String(err.Error()))
	} else {
		next.Push(clos)
	}
	return next, nil
}
