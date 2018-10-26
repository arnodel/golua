package ast

func NewFunctionStat(fName Var, method Name, fx Function) AssignStat {
	// TODO: include the "function" keywork in the location calculation
	if method.Val != "" {
		loc := fx.Locate()
		fx = NewFunction(
			nil, nil,
			ParList{append([]Name{{Val: "self"}}, fx.Params...), fx.HasDots},
			fx.Body,
		)
		fx.Location = loc
		fx.Name = method.FunctionName()
		fName = NewIndexExp(fName, method.AstString())
	} else {
		fx.Name = fName.FunctionName()
	}
	return NewAssignStat([]Var{fName}, []ExpNode{fx})
}
