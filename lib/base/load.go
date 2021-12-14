package base

import (
	"bytes"
	"strings"

	rt "github.com/arnodel/golua/runtime"
	"github.com/arnodel/golua/scanner"
)

const maxChunkNameLen = 59

func load(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	var (
		chunk     []byte
		chunkName = "chunk"
		chunkMode = "bt"
		chunkEnv  = rt.TableValue(t.GlobalEnv())
		next      = c.Next()
	)

	switch nargs := c.NArgs(); {
	case nargs >= 4:
		chunkEnv = c.Arg(3)
		fallthrough
	case nargs >= 3:
		if !c.Arg(2).IsNil() {
			mode, err := c.StringArg(2)
			if err != nil {
				return nil, err
			}
			chunkMode = string(mode)
		}
		fallthrough
	case nargs >= 2:
		if !c.Arg(1).IsNil() {
			name, err := c.StringArg(1)
			if err != nil {
				return nil, err
			}
			chunkName = name
			if len(name) > maxChunkNameLen {
				chunkName = chunkName[:maxChunkNameLen]
			}
		}
		fallthrough
	case nargs >= 1:
		switch x := c.Arg(0); x.Type() {
		case rt.StringType:
			xs := x.AsString()
			// Use CPU as well as memory, but not much
			t.LinearRequire(10, uint64(len(xs)))
			chunk = []byte(xs)
		case rt.FunctionType:
			var buf bytes.Buffer
			for {
				bit, err := rt.Call1(t, x)
				if err != nil {
					t.Push1(next, rt.NilValue)
					t.Push1(next, rt.StringValue(err.Error()))
					t.ReleaseBytes(buf.Len())
					return next, nil
				}
				if bit.IsNil() {
					break
				}
				bitString, ok := bit.TryString()
				if !ok {
					t.Push(next, rt.NilValue, rt.StringValue("reader must return a string"))
					t.ReleaseBytes(buf.Len())
					return next, nil
				}
				if len(bitString) == 0 {
					break
				}
				// When bitString is small, cpu required may be 0 but thats' ok
				// because cpu was used when calling the function.
				t.LinearRequire(10, uint64(len(bitString)))
				buf.WriteString(bitString)
			}
			chunk = buf.Bytes()
		default:
			return nil, rt.NewErrorS("#1 must be a string or function")
		}
	}
	// The chunk is no longer used once we leave this function.
	defer t.ReleaseBytes(len(chunk))

	canBeBinary := strings.IndexByte(chunkMode, 'b') >= 0
	canBeText := strings.IndexByte(chunkMode, 't') >= 0

	if canBeBinary {
		r := bytes.NewBuffer(chunk)
		k, used, err := rt.UnmarshalConst(r, t.LinearUnused(10))
		t.LinearRequire(10, used)
		if err != rt.ErrInvalidMarshalPrefix {
			if err != nil {
				return nil, rt.NewErrorE(err)
			}
			code, ok := k.TryCode()
			if !ok {
				return nil, rt.NewErrorF("Expected function to load")
			}
			clos := rt.NewClosure(t.Runtime, code)
			if code.UpvalueCount > 0 {
				clos.AddUpvalue(rt.NewCell(chunkEnv))
				t.RequireCPU(uint64(code.UpvalueCount))
				for i := int16(1); i < code.UpvalueCount; i++ {
					clos.AddUpvalue(rt.NewCell(rt.NilValue))
				}
			}
			return c.PushingNext(t.Runtime, rt.FunctionValue(clos)), nil
		}
	} else if rt.HasMarshalPrefix(chunk) {
		t.Push(next, rt.NilValue, rt.StringValue("attempt to load a binary chunk"))
	}

	if !canBeText {
		t.Push(next, rt.NilValue, rt.StringValue("attempt to load a text chunk"))
		return next, nil
	}
	clos, err := t.CompileAndLoadLuaChunk(chunkName, chunk, chunkEnv, scanner.NoSpecialComment())
	if err != nil {
		t.Push(next, rt.NilValue, rt.StringValue(err.Error()))
	} else {
		t.Push1(next, rt.FunctionValue(clos))
	}
	return next, nil
}
