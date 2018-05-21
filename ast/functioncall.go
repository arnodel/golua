package ast

import "github.com/arnodel/golua/ir"

type FunctionCall struct {
	Location
	target ExpNode
	method Name
	args   []ExpNode
}

func NewFunctionCall(target ExpNode, method Name, args []ExpNode) (*FunctionCall, error) {
	// TODO: fix this by creating an Args node
	loc := target.Locate()
	if len(args) > 0 {
		loc = MergeLocations(loc, args[len(args)-1])
	} else if method.string != "" {
		loc = MergeLocations(loc, method)
	}
	return &FunctionCall{
		Location: loc,
		target:   target,
		method:   method,
		args:     args,
	}, nil
}

func (f FunctionCall) HWrite(w HWriter) {
	w.Writef("call")
	w.Indent()
	w.Next()
	w.Writef("target: ")
	// w.Indent()
	f.target.HWrite(w)
	// w.Dedent()
	if f.method.string != "" {
		w.Next()
		w.Writef("method: %s", f.method)
	}
	for i, arg := range f.args {
		w.Next()
		w.Writef("arg_%d: ", i)
		arg.HWrite(w)
	}
	w.Dedent()
}

func (f FunctionCall) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	f.CompileCall(c, false)
	EmitInstr(c, f, ir.Receive{Dst: []ir.Register{dst}})
	return dst
}

// TODO: move this to somewhere better
func CallWithArgs(c *ir.Compiler, args []ExpNode, contReg ir.Register) {
	c.TakeRegister(contReg)
	for i, arg := range args {
		var argReg ir.Register
		if i == len(args)-1 {
			switch x := arg.(type) {
			case *FunctionCall:
				x.CompileCall(c, false)
				argReg = c.GetFreeRegister()
				EmitInstr(c, arg, ir.ReceiveEtc{Etc: argReg})
				EmitInstr(c, arg, ir.Push{Cont: contReg, Item: argReg, Etc: true})
				continue
			case EtcType:
				argReg, ok := c.GetRegister("...")
				if !ok {
					panic("etc not defined")
				}
				EmitInstr(c, arg, ir.Push{Cont: contReg, Item: argReg, Etc: true})
				continue
			}
		}
		argReg = CompileExp(c, arg)
		EmitInstr(c, arg, ir.Push{Cont: contReg, Item: argReg})
	}
	c.EmitNoLine(ir.Call{Cont: contReg})
	c.ReleaseRegister(contReg)
}

func (f FunctionCall) CompileCall(c *ir.Compiler, tail bool) {
	fReg := CompileExp(c, f.target)
	var contReg ir.Register
	if f.method.string != "" {
		self := fReg
		c.TakeRegister(self)
		fReg = c.GetFreeRegister()
		mReg := c.GetFreeRegister()
		EmitLoadConst(c, f.method, ir.String(f.method.string), mReg)
		EmitInstr(c, f.target, ir.Lookup{
			Dst:   fReg,
			Table: self,
			Index: mReg,
		})
		contReg = c.GetFreeRegister()
		EmitInstr(c, f, ir.MkCont{Dst: contReg, Closure: fReg, Tail: tail})
		EmitInstr(c, f, ir.Push{Cont: fReg, Item: self})
		c.ReleaseRegister(self)
	} else {
		contReg = c.GetFreeRegister()
		EmitInstr(c, f, ir.MkCont{Dst: contReg, Closure: fReg, Tail: tail})
	}
	CallWithArgs(c, f.args, contReg)
}

func (f FunctionCall) CompileStat(c *ir.Compiler) {
	f.CompileCall(c, false)
	EmitInstr(c, f, ir.Receive{})
}
