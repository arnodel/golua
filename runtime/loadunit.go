package runtime

import (
	"io"

	"github.com/arnodel/golua/code"
)

// Code represents the code for a Lua function together with all the constants
// that this function uses.  It can be turned into a closure by adding upvalues.
type Code struct {
	source, name string
	code         []code.Opcode
	lines        []int32
	consts       []Konst
	UpvalueCount int16
	UpNames      []string
	RegCount     int16
	CellCount    int16
}

// RefactorConsts returns an equivalent *Code this consts "refactored", which
// means that the consts are slimmed down to only contains the constants
// required for the function.
func (c *Code) RefactorConsts() *Code {
	opcodes := make([]code.Opcode, len(c.code))
	var consts []Konst
	constMap := map[code.KIndex]code.KIndex{}
	for i, op := range c.code {
		if op.TypePfx() == code.Type3Pfx {
			unop := op.GetY()
			if unop.LoadsK() {
				// We are loading a constant
				n := op.GetKIndex()
				m, ok := constMap[n]
				if !ok {
					m = code.KIndexFromInt(len(consts))
					constMap[n] = m
					newConst := c.consts[n]
					if unop == code.OpClosureK {
						// It's a closure so we need to refactor its consts
						newConst = newConst.(*Code).RefactorConsts()
					}
					consts = append(consts, newConst)
				}
				op = op.SetKIndex(m)
			}
		}
		opcodes[i] = op
	}
	cc := *c
	cc.code = opcodes
	cc.consts = consts
	return &cc
}

func (c *Code) writeKonst(w io.Writer) (err error) {
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
	bwrite(w, c.CellCount)
	bwrite(w, int64(len(c.UpNames)))
	for _, n := range c.UpNames {
		swrite(w, n)
	}
	return
}

func (c *Code) loadKonst(r io.Reader) (err error) {
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
	c.consts = make([]Konst, sz)
	for i := range c.consts {
		c.consts[i], err = LoadConst(r)
		if err != nil {
			return
		}
	}
	bread(r, &c.UpvalueCount)
	bread(r, &c.RegCount)
	bread(r, &c.CellCount)
	bread(r, &sz)
	c.UpNames = make([]string, sz)
	for i := range c.UpNames {
		sread(r, &c.UpNames[i])
	}
	return
}

// LoadLuaUnit turns a code unit into a closure given an environment env.
func LoadLuaUnit(unit *code.Unit, env Value) *Closure {
	constants := make([]Konst, len(unit.Constants))
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
				UpNames:      k.UpNames,
				RegCount:     k.RegCount,
				CellCount:    k.CellCount,
			}
		default:
			panic("Unsupported constant type")
		}
	}
	mainCode := constants[0].(*Code) // It must be some code
	clos := NewClosure(mainCode)
	if mainCode.UpvalueCount > 0 {
		clos.AddUpvalue(Cell{&env})
	}
	return clos
}
