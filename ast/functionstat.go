package ast

// NewFunctionStat returns an AssgnStat, ie. "function f() ..." gets transformed
// to "f = function() ...".  This is a shortcut, probably should make a specific
// node for this.
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
