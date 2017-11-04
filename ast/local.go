package ast

type LocalStat struct {
	names  []Name
	values []ExpNode
}

func (s LocalStat) HWrite(w HWriter) {
	w.Writef("local")
	w.Indent()
	for i, name := range s.names {
		w.Next()
		w.Writef("name_%d: %s", i, name)
	}
	for i, val := range s.values {
		w.Next()
		w.Writef("val_%d: ", i)
		val.HWrite(w)
	}
	w.Dedent()
}

func (s LocalStat) CompileStat(c *Compiler) {
	// nameCount := len(s.names)
	// valueCount := len(s.values)
	// commonCount := nameCount
	// if commonCount > valueCount {
	// 	commonCount = valueCount
	// }
	// if nameCount < valueCount {
	// 	f, ok := s.values[valueCount-1]
	// }
	// valueRegs := []ir.Register{}
	// for i := 0; i < len(s.values)-1; i++ {
	// 	val := s.values[i]
	// 	valueRegs = append(valueRegs, c.CompileExp(val))
	// }
	// if len(s.values) < len(s.names) {
	// 	f, ok := s.values[len(s.values)-1].(FunctionCall)
	// 	if ok {
	// 		var fRegs []ir.Register
	// 		for i := len(values) - 1; i < len(s.names); i++ {

	// 			fRegs = append(fRegs, c.NewRegister())
	// 		}
	// 	}
	// }
}

func NewLocalStat(names []Name, values []ExpNode) (LocalStat, error) {
	return LocalStat{names: names, values: values}, nil
}
