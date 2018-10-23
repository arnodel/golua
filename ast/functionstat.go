package ast

func NewFunctionStat(fName Var, method Name, fx Function) AssignStat {
	// TODO: include the "function" keywork in the location calculation
	if method.string != "" {
		loc := fx.Locate()
		fx = NewFunction(
			nil, nil,
			ParList{append([]Name{{string: "self"}}, fx.params...), fx.hasDots},
			fx.body,
		)
		fx.Location = loc
		fx.name = method.FunctionName()
		fName = NewIndexExp(fName, String{val: []byte(method.string)})
	} else {
		fx.name = fName.FunctionName()
	}
	return NewAssignStat([]Var{fName}, []ExpNode{fx})
}
