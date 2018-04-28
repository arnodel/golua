package runtime

import "github.com/arnodel/golua/code"

type Code struct {
	code         []code.Opcode
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
			constants[i] = NilType{}
		case code.Code:
			constants[i] = &Code{
				code:         unit.Code[k.StartOffset:k.EndOffset],
				consts:       constants,
				UpvalueCount: k.UpvalueCount,
				RegCount:     k.RegCount,
			}
		default:
			panic("Unsupported constant type")
		}
	}
	clos := NewClosure(constants[0].(*Code))
	clos.AddUpvalue(env)
	return clos
}
