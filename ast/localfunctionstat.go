package ast

type LocalFunctionStat struct {
	Location
	Function
	Name Name
}

var _ Stat = LocalFunctionStat{}

func NewLocalFunctionStat(name Name, fx Function) LocalFunctionStat {
	fx.Name = name.Val
	return LocalFunctionStat{
		Location: MergeLocations(name, fx), // TODO: use "local" for location start
		Function: fx,
		Name:     name,
	}
}

func (s LocalFunctionStat) ProcessStat(p StatProcessor) {
	p.ProcessLocalFunctionStat(s)
}

func (s LocalFunctionStat) HWrite(w HWriter) {
	w.Writef("local function ")
	s.Name.HWrite(w)
	s.Function.HWrite(w)
}
