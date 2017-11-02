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
	// TODO
}

func NewLocalStat(names []Name, values []ExpNode) (LocalStat, error) {
	return LocalStat{names: names, values: values}, nil
}
