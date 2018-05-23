package runtime

import (
	"github.com/arnodel/golua/code"
)

type Code struct {
	source, name string
	code         []code.Opcode
	lines        []int
	consts       []Value
	UpvalueCount int
	RegCount     int
}

func LoadLuaUnit(unit *code.Unit, env *Table) *Closure {
	constants := make([]Value, len(unit.Constants))
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
			// constants[i] = NilType{}
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
