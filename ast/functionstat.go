package ast

func NewFunctionStat(name FunctionName, fx Function) (AssignStat, error) {
	fName := name.name
	if name.method != "" {
		fx, _ = NewFunction(
			ParList{append([]Name{"self"}, fx.params...), fx.hasDots},
			fx.body,
		)
		fName, _ = NewIndexExp(name.name, String(name.method))
	}
	return NewAssignStat([]Var{fName}, []ExpNode{fx})
}

type FunctionName struct {
	name   Var
	method Name
}

func NewFunctionName(name Var, method Name) (FunctionName, error) {
	return FunctionName{
		name:   name,
		method: method,
	}, nil
}

func (n FunctionName) HWrite(w HWriter) {
	n.name.HWrite(w)
	if n.method != "" {
		w.Writef(":%s", n.method)
	}
}
