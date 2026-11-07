package ir

// A Pass rewrites a function's code, keeping its meaning.  Passes run on the
// IR between the AST compiler and the code generator; notes/ir-passes.md says
// what a pass may assume.  They need not keep TakeRegister and
// ReleaseRegister right: the code generator computes register lifetimes
// itself.
type Pass func(Code) Code

// Optimise applies the passes, in order, to every function in consts.
func Optimise(consts []Constant, passes ...Pass) []Constant {
	out := make([]Constant, len(consts))
	for i, k := range consts {
		if c, ok := k.(*Code); ok {
			oc := *c
			for _, pass := range passes {
				oc = pass(oc)
			}
			out[i] = &oc
		} else {
			out[i] = k
		}
	}
	return out
}
