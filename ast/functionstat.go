package ast

func NewFunctionStat(fName Var, method Name, fx Function) AssignStat {
	// TODO: include the "function" keywork in the location calculation
	if method.Val != "" {
		loc := fx.Locate()
		fx = NewFunction(
			nil, nil,
			ParList{append([]Name{{Val: "self"}}, fx.params...), fx.hasDots},
			fx.body,
		)
		fx.Location = loc
		fx.name = method.FunctionName()
		fName = NewIndexExp(fName, String{Val: []byte(method.Val)})
	} else {
		fx.name = fName.FunctionName()
	}
	return NewAssignStat([]Var{fName}, []ExpNode{fx})
}
