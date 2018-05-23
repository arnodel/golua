package ast

func NewFunctionStat(name FunctionName, fx Function) (AssignStat, error) {
	// TODO: include the "function" keywork in the location calculation
	fName := name.name
	if name.method.string != "" {
		loc := fx.Locate()
		fx, _ = NewFunction(
			nil, nil,
			ParList{append([]Name{{string: "self"}}, fx.params...), fx.hasDots},
			fx.body,
		)
		fx.Location = loc
		fx.name = name.method.FunctionName()
		fName, _ = NewIndexExp(name.name, String{val: []byte(name.method.string)})
	} else {
		fx.name = fName.FunctionName()
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
