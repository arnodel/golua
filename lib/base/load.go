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
	var (
		chunk     []byte
		chunkName = "chunk"
		chunkMode = "bt"
		chunkEnv  = t.GlobalEnv()
		next      = c.Next()
	)

	switch nargs := c.NArgs(); {
	case nargs >= 4:
		var err *rt.Error
		chunkEnv, err = c.TableArg(3)
		if err != nil {
			return nil, err.AddContext(c)
		}
		fallthrough
	case nargs >= 3:
		if !c.Arg(2).IsNil() {
			mode, err := c.StringArg(2)
			if err != nil {
				return nil, err.AddContext(c)
			}
			chunkMode = string(mode)
		}
		fallthrough
	case nargs >= 2:
		if !c.Arg(1).IsNil() {
			name, err := c.StringArg(1)
			if err != nil {
				return nil, err.AddContext(c)
			}
			chunkName = string(name)
		}
		fallthrough
	case nargs >= 1:
		switch x := c.Arg(0); x.Type() {
		case rt.StringType:
			chunk = []byte(x.AsString())
		case rt.FunctionType:
			var buf bytes.Buffer
			for {
				bit, err := rt.Call1(t, x)
				if err != nil {
					next.Push(rt.NilValue)
					next.Push(rt.StringValue(err.Error()))
					return next, nil
				}
				if bit.IsNil() {
					break
				}
				bitString, ok := bit.TryString()
				if !ok {
					rt.Push(next, rt.NilValue, rt.StringValue("reader must return a string"))
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
			rt.Push(next, rt.NilValue, rt.StringValue("attempt to load a binary chunk"))
			return next, nil
		}
		r := bytes.NewBuffer(chunk)
		k, err := rt.LoadConst(r)
		if err != nil {
			return nil, rt.NewErrorE(err).AddContext(c)
		}
		code, ok := k.Value().TryCode()
		if !ok {
			return nil, rt.NewErrorF("Expected function to load").AddContext(c)
		}
		clos := rt.NewClosure(code)
		if code.UpvalueCount > 0 {
			envVal := rt.TableValue(chunkEnv)
			clos.AddUpvalue(rt.NewCell(envVal))
			for i := int16(1); i < code.UpvalueCount; i++ {
				clos.AddUpvalue(rt.NewCell(rt.NilValue))
			}
		}
		return c.PushingNext(rt.FunctionValue(clos)), nil
	} else if !canBeText {
		rt.Push(next, rt.NilValue, rt.StringValue("attempt to load a text chunk"))
		return next, nil
	}
	clos, err := rt.CompileAndLoadLuaChunk(chunkName, chunk, chunkEnv)
	if err != nil {
		rt.Push(next, rt.NilValue, rt.StringValue(err.Error()))
	} else {
		next.Push(rt.FunctionValue(clos))
	}
	return next, nil
}
