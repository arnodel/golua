package runtime

import (
	"unsafe"

	"github.com/arnodel/golua/code"
)

// Code represents the code for a Lua function together with all the constants
// that this function uses.  It can be turned into a closure by adding upvalues.
type Code struct {
	source, name string
	code         []code.Opcode
	lines        []int32
	consts       []Value
	UpvalueCount int16
	UpNames      []string
	RegCount     int16
	CellCount    int16
}

// RefactorConsts returns an equivalent *Code this consts "refactored", which
// means that the consts are slimmed down to only contains the constants
// required for the function.
func (r *Runtime) RefactorCodeConsts(c *Code) *Code {
	r.RequireArrSize(unsafe.Sizeof(code.Opcode(0)), len(c.code))
	opcodes := make([]code.Opcode, len(c.code))
	var consts []Value
	constMap := map[code.KIndex]code.KIndex{}

	// Require CPU for the loop below
	r.RequireCPU(uint64(len(c.code)))

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
						newConst = CodeValue(r.RefactorCodeConsts(newConst.AsCode()))
					}
					r.RequireSize(unsafe.Sizeof(Value{}))
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

// LoadLuaUnit turns a code unit into a closure given an environment env.
func (r *Runtime) LoadLuaUnit(unit *code.Unit, env Value) *Closure {
	r.RequireArrSize(unsafe.Sizeof(Value{}), len(unit.Constants))
	constants := make([]Value, len(unit.Constants))

	// Require memory for all the code at once, rather than in bits in the
	// code.Code case below
	r.RequireArrSize(unsafe.Sizeof(code.Opcode(0)), len(unit.Code))
	r.RequireArrSize(4, len(unit.Lines))

	// Require CPU for the loop below
	r.RequireCPU(uint64(len(unit.Constants)))

	for i, ck := range unit.Constants {
		switch k := ck.(type) {
		case code.Int:
			constants[i] = IntValue(int64(k))
		case code.Float:
			constants[i] = FloatValue(float64(k))
		case code.String:
			// The strings are already accounted for memory-wise
			constants[i] = StringValue(string(k))
		case code.Bool:
			constants[i] = BoolValue(bool(k))
		case code.NilType:
			// Do nothing as constants[i] == nil
		case code.Code:
			r.RequireSize(unsafe.Sizeof(Code{}))
			var lines []int32
			if unit.Lines != nil {
				lines = unit.Lines[k.StartOffset:k.EndOffset]
			}
			constants[i] = CodeValue(&Code{
				source:       unit.Source,
				name:         k.Name,
				code:         unit.Code[k.StartOffset:k.EndOffset],
				lines:        lines,
				consts:       constants,
				UpvalueCount: k.UpvalueCount,
				UpNames:      k.UpNames,
				RegCount:     k.RegCount,
				CellCount:    k.CellCount,
			})
		default:
			panic("Unsupported constant type")
		}
	}
	mainCode := constants[0].AsCode() // It must be some code
	clos := NewClosure(r, mainCode)
	if mainCode.UpvalueCount > 0 {
		clos.AddUpvalue(Cell{&env})
	}
	return clos
}
