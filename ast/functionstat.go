package ast

func NewFunctionStat(name FunctionName, fx Function) (AssignStat, error) {
	fName := name.name
	if name.method.string != "" {
		fx, _ = NewFunction(
			ParList{append([]Name{{string: "self"}}, fx.params...), fx.hasDots},
			fx.body,
		)
		fName, _ = NewIndexExp(name.name, String{val: []byte(name.method.string)})
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
	if n.method.string != "" {
		w.Writef(":%s", n.method)
	}
}
