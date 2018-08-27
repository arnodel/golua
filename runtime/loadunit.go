package runtime

import (
	"io"

	"github.com/arnodel/golua/code"
)

type Code struct {
	source, name string
	code         []code.Opcode
	lines        []int32
	consts       []Const
	UpvalueCount int16
	RegCount     int16
}

func (c *Code) RefactorConsts() *Code {
	opcodes := make([]code.Opcode, len(c.code))
	var consts []Const
	constMap := map[uint16]uint16{}
	for i, op := range c.code {
		if op.TypePfx() == code.Type3Pfx {
			// We are loading a constant
			n := op.GetN()
			m, ok := constMap[n]
			if !ok {
				m = uint16(len(consts))
				constMap[n] = m
				newConst := c.consts[n]
				if code.UnOpK16(op.GetY()) == code.OpClosureK {
					// It's a closure so we need to refactor its consts
					newConst = newConst.(*Code).RefactorConsts()
				}
				consts = append(consts, newConst)
			}
			op = op.SetN(m)
		}
		opcodes[i] = op
	}
	cc := *c
	cc.code = opcodes
	cc.consts = consts
	return &cc
}

func (c *Code) WriteConst(w io.Writer) (err error) {
	_, err = w.Write([]byte{constTypeCode})
	if err != nil {
		return
	}
	swrite(w, c.source)
	swrite(w, c.name)
	bwrite(w, int64(len(c.code)))
	for _, opcode := range c.code {
		bwrite(w, int32(opcode))
	}
	bwrite(w, int64(len(c.lines)))
	bwrite(w, c.lines)
	bwrite(w, int64(len(c.consts)))
	for _, k := range c.consts {
		WriteConst(w, k)
	}
	bwrite(w, c.UpvalueCount)
	bwrite(w, c.RegCount)
	return
}

func (c *Code) LoadConst(r io.Reader) (err error) {
	sread(r, &c.source)
	sread(r, &c.name)
	var sz int64
	bread(r, &sz)
	c.code = make([]code.Opcode, sz)
	for i := range c.code {
		var op int32
		bread(r, &op)
		c.code[i] = code.Opcode(op)
	}
	bread(r, &sz)
	c.lines = make([]int32, sz)
	bread(r, c.lines)
	bread(r, &sz)
	c.consts = make([]Const, sz)
	for i := range c.consts {
		c.consts[i], err = LoadConst(r)
		if err != nil {
			return
		}
	}
	bread(r, &c.UpvalueCount)
	bread(r, &c.RegCount)
	return
}

func LoadLuaUnit(unit *code.Unit, env *Table) *Closure {
	constants := make([]Const, len(unit.Constants))
	for i, ck := range unit.Constants {
		switch k := ck.(type) {
		case code.Int:
			constants[i] = Int(k)
		case code.Float:
			constants[i] = Float(k)
		case code.String:
			constants[i] = String(k)
		case code.Bool:
			constants[i] = Bool(k)
		case code.NilType:
			// Do nothing as constants[i] == nil
		case code.Code:
			constants[i] = &Code{
				source:       unit.Source,
				name:         k.Name,
				code:         unit.Code[k.StartOffset:k.EndOffset],
				lines:        unit.Lines[k.StartOffset:k.EndOffset],
				consts:       constants,
				UpvalueCount: k.UpvalueCount,
				RegCount:     k.RegCount,
			}
		default:
			panic("Unsupported constant type")
		}
	}
	var envVal Value = env
	mainCode := constants[0].(*Code) // It must be some code
	clos := NewClosure(mainCode)
	if mainCode.UpvalueCount > 0 {
		clos.AddUpvalue(Cell{&envVal})
	}
	return clos
}
