package ast

import "github.com/arnodel/golua/ir"

type Stat interface {
	Node
	CompileStat(c *Compiler)
}

type BreakStat struct{}

func (s BreakStat) HWrite(w HWriter) {
	w.Writef("break")
}

func (s BreakStat) CompileStat(c *Compiler) {
	// TODO
}

type AssignStat struct {
	dst []Var
	src []ExpNode
}

func (s AssignStat) HWrite(w HWriter) {
	w.Writef("assign")
	w.Indent()
	for i, v := range s.dst {
		w.Next()
		w.Writef("dst_%d: ", i)
		v.HWrite(w)
	}
	for i, n := range s.src {
		w.Next()
		w.Writef("src_%d: ", i)
		n.HWrite(w)
	}
	w.Dedent()
}

func CompileExpList(c *Compiler, exps []ExpNode, dstRegs []ir.Register) {
	commonCount := len(exps)
	if commonCount > len(dstRegs) {
		commonCount = len(dstRegs)
	}
	var fCall FunctionCall
	doFCall := false
	if len(exps) < len(dstRegs) && len(exps) > 0 {
		fCall, doFCall = exps[len(exps)-1].(FunctionCall)
		if doFCall {
			commonCount--
		}
	}
	for i, exp := range exps[:commonCount] {
		dst := c.GetFreeRegister()
		reg := exp.CompileExp(c, dst)
		EmmitMove(c, dst, reg)
		c.TakeRegister(dst)
		dstRegs[i] = dst
	}
	for i := commonCount; i < len(dstRegs); i++ {
		dst := c.GetFreeRegister()
		c.TakeRegister(dst)
		dstRegs[i] = dst
	}
	if doFCall {
		fCall.CompileCall(c)
		c.Emit(ir.Receive{Dst: dstRegs[commonCount:]})
	} else if len(dstRegs) > len(exps) {
		nilK := ir.NilType{}
		for _, dst := range dstRegs[len(exps):] {
			EmitConstant(c, nilK, dst)
		}
	}
}

func (s AssignStat) CompileStat(c *Compiler) {
	resultRegs := make([]ir.Register, len(s.dst))
	CompileExpList(c, s.src, resultRegs)
	for i, reg := range resultRegs {
		c.ReleaseRegister(reg)
		s.dst[i].CompileAssign(c, reg)
	}
}

type GotoStat struct {
	label Name
}

func (s GotoStat) HWrite(w HWriter) {
	w.Writef("goto %s", s.label)
}

func (s GotoStat) CompileStat(c *Compiler) {
	// TODO
}

type BlockStat struct {
	statements   []Stat
	returnValues []ExpNode
}

func (s BlockStat) HWrite(w HWriter) {
	w.Writef("block")
	w.Indent()
	for _, stat := range s.statements {
		w.Next()
		stat.HWrite(w)
	}
	if s.returnValues != nil {
		w.Next()
		w.Writef("return")
		w.Indent()
		for _, val := range s.returnValues {
			w.Next()
			val.HWrite(w)
		}
		w.Dedent()
	}
	w.Dedent()
}

func (s BlockStat) CompileStat(c *Compiler) {
	c.PushContext()
	for _, stat := range s.statements {
		stat.CompileStat(c)
	}
	if s.returnValues != nil {
		cont, ok := c.GetRegister(Name("<caller>"))
		if !ok {
			panic("Cannot return: no caller")
		}
		CallWithArgs(c, s.returnValues, cont)
	}
	c.PopContext()
}

type CondStat struct {
	cond ExpNode
	body Stat
}

func (s CondStat) HWrite(w HWriter) {
	s.cond.HWrite(w)
	w.Next()
	w.Writef("body: ")
	s.body.HWrite(w)
}

func (s CondStat) CompileStat(c *Compiler) {
	// TODO
}

type WhileStat struct {
	CondStat
}

func (s WhileStat) HWrite(w HWriter) {
	w.Writef("while: ")
	s.CondStat.HWrite(w)
}

func (s WhileStat) CompileStat(c *Compiler) {
	// TODO
}

type RepeatStat struct {
	CondStat
}

func (s RepeatStat) HWrite(w HWriter) {
	w.Writef("repeat if: ")
	s.CondStat.HWrite(w)
}

type IfStat struct {
	ifstat      CondStat
	elseifstats []CondStat
	elsestat    Stat
}

func (s IfStat) HWrite(w HWriter) {
	w.Writef("if: ")
	w.Indent()
	s.ifstat.HWrite(w)
	for _, elseifstat := range s.elseifstats {
		w.Next()
		w.Writef("elseif: ")
		elseifstat.HWrite(w)
	}
	if s.elsestat != nil {
		w.Next()
		w.Writef("else:")
		s.elsestat.HWrite(w)
	}
	w.Dedent()
}

func (s IfStat) CompileStat(c *Compiler) {
	// TODO
}

type ForStat struct {
	itervar Name
	start   ExpNode
	stop    ExpNode
	step    ExpNode
	body    Stat
}

func (s *ForStat) HWrite(w HWriter) {
	w.Writef("for %s", s.itervar)
	w.Indent()
	if s.start != nil {
		w.Next()
		w.Writef("start: ")
		s.start.HWrite(w)
	}
	if s.stop != nil {
		w.Next()
		w.Writef("stop: ")
		s.stop.HWrite(w)
	}
	if s.step != nil {
		w.Next()
		w.Writef("step: ")
		s.step.HWrite(w)
	}
	w.Next()
	s.body.HWrite(w)
	w.Dedent()
}

func (s ForStat) CompileStat(c *Compiler) {
	// TODO
}

type ForInStat struct {
	itervars []Name
	params   []ExpNode
	body     Stat
}

func (s *ForInStat) HWrite(w HWriter) {
	w.Writef("for in")
	w.Indent()
	for i, v := range s.itervars {
		w.Next()
		w.Writef("var_%d: ", i)
		v.HWrite(w)
	}
	for i, p := range s.params {
		w.Next()
		w.Writef("param_%d", i)
		p.HWrite(w)
	}
	w.Next()
	w.Writef("body: ")
	s.body.HWrite(w)
	w.Dedent()
}

func (s ForInStat) CompileStat(c *Compiler) {
	// TODO
}

type LabelStat Name

func (s LabelStat) HWrite(w HWriter) {
	w.Writef("label %s", string(s))
}

func (s LabelStat) CompileStat(c *Compiler) {
	// TODO
}

type EmptyStat struct{}

func (s EmptyStat) HWrite(w HWriter) {
	w.Writef("empty stat")
}

func (s EmptyStat) CompileStat(c *Compiler) {
	// TODO
}

func NewAssignStat(dst []Var, src []ExpNode) (AssignStat, error) {
	return AssignStat{
		dst: dst,
		src: src,
	}, nil
}

func NewBreakStat() (BreakStat, error) {
	return BreakStat{}, nil
}

func NewGotoStat(lbl Name) (GotoStat, error) {
	return GotoStat{label: lbl}, nil
}

func NewBlockStat(stats []Stat, rtn []ExpNode) (BlockStat, error) {
	return BlockStat{stats, rtn}, nil
}

func NewWhileStat(cond ExpNode, body Stat) (WhileStat, error) {
	return WhileStat{CondStat{cond: cond, body: body}}, nil
}

func NewRepeatStat(body Stat, cond ExpNode) (RepeatStat, error) {
	return RepeatStat{CondStat{body: body, cond: cond}}, nil
}

func NewIfStat() IfStat {
	return IfStat{}
}

func (s IfStat) AddIf(cond ExpNode, body Stat) (IfStat, error) {
	s.ifstat = CondStat{cond, body}
	return s, nil
}

func (s IfStat) AddElse(body Stat) (IfStat, error) {
	s.elsestat = body
	return s, nil
}

func (s IfStat) AddElseIf(cond ExpNode, body Stat) (IfStat, error) {
	s.elseifstats = append(s.elseifstats, CondStat{cond, body})
	return s, nil
}

func NewForStat(itervar Name, params []ExpNode, body Stat) (*ForStat, error) {
	return &ForStat{
		itervar: itervar,
		start:   params[0],
		stop:    params[1],
		step:    params[2],
		body:    body,
	}, nil
}

func NewForInStat(itervars []Name, params []ExpNode, body Stat) (ForInStat, error) {
	return ForInStat{
		itervars: itervars,
		params:   params,
		body:     body,
	}, nil
}

func NewLabelStat(label Name) (LabelStat, error) {
	return LabelStat(label), nil
}

func NewEmptyStat() (EmptyStat, error) {
	return EmptyStat{}, nil
}
