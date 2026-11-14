package ircomp

import (
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/arnodel/golua/code"
	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/ops"
)

type instrCompiler struct {
	*ConstantCompiler
	*regAllocator
	line int
}

var _ ir.InstrProcessor = instrCompiler{}

func (ic instrCompiler) Emit(opcode code.Opcode) {
	ic.builder.Emit(opcode, ic.line)
	ic.atEntry = false
}

func (ic instrCompiler) EmitJump(opcode code.Opcode, lbl code.Label) {
	ic.builder.EmitJump(opcode, lbl, ic.line)
}

// ProcessCombineInstr compiles a Combine instruction.
func (ic instrCompiler) ProcessCombineInstr(c ir.Combine) {
	codeOp, ok := codeBinOp[c.Op]
	if !ok {
		panic(fmt.Sprintf("Cannot compile %v: invalid op", c))
	}
	opcode := code.Combine(codeOp, ic.codeReg(c.Dst), ic.codeReg(c.Lsrc), ic.codeReg(c.Rsrc))
	ic.Emit(opcode)
}

var codeBinOp = map[ops.Op]code.BinOp{
	ops.OpLt:       code.OpLt,
	ops.OpLeq:      code.OpLeq,
	ops.OpEq:       code.OpEq,
	ops.OpBitOr:    code.OpBitOr,
	ops.OpBitXor:   code.OpBitXor,
	ops.OpBitAnd:   code.OpBitAnd,
	ops.OpShiftL:   code.OpShiftL,
	ops.OpShiftR:   code.OpShiftR,
	ops.OpConcat:   code.OpConcat,
	ops.OpAdd:      code.OpAdd,
	ops.OpSub:      code.OpSub,
	ops.OpMul:      code.OpMul,
	ops.OpDiv:      code.OpDiv,
	ops.OpFloorDiv: code.OpFloorDiv,
	ops.OpMod:      code.OpMod,
	ops.OpPow:      code.OpPow,
}

// ProcessTransformInstr compiles a Transform instruction.
func (ic instrCompiler) ProcessTransformInstr(t ir.Transform) {
	codeOp, ok := codeUnOp[t.Op]
	if !ok {
		panic(fmt.Sprintf("Cannot compile %v: invalid op", t))
	}
	dst, src := ic.codeReg(t.Dst), ic.codeReg(t.Src)
	if t.Op == ops.OpId && dst == src {
		// A move between two IR registers given the same slot does nothing.
		return
	}
	ic.Emit(code.Transform(codeOp, dst, src))
}

// ProcessLoadInstr compiles a Load instruction.
func (ic instrCompiler) ProcessLoadInstr(l ir.Load) {
	dst, box := ic.codeReg(l.Dst), ic.codeReg(l.Box)
	if up, isUp := ic.upvalueOf[l.Box]; isUp && ic.atEntry && ic.registers[l.Box].IsFixed {
		// A load of a fixed upvalue at the function's entry, before anything
		// but the parameters' recv: the frame does it when it is made, so no
		// opcode.  Nothing can change a fixed upvalue in between.
		ic.entryLoads = append(ic.entryLoads, code.EntryLoad{Up: uint8(up), Reg: dst.Idx()})
		return
	}
	ic.Emit(code.Load(dst, box))
}

// ProcessStoreInstr compiles a Store instruction.
func (ic instrCompiler) ProcessStoreInstr(s ir.Store) {
	ic.Emit(code.Store(ic.codeReg(s.Box), ic.codeReg(s.Src)))
}

// ProcessNewBoxInstr compiles a NewBox instruction.
func (ic instrCompiler) ProcessNewBoxInstr(n ir.NewBox) {
	ic.Emit(code.NewBox(ic.codeReg(n.Dst)))
}

var codeUnOp = map[ops.Op]code.UnOp{
	ops.OpNeg:    code.OpNeg,
	ops.OpNot:    code.OpNot,
	ops.OpLen:    code.OpLen,
	ops.OpBitNot: code.OpBitNot,
	ops.OpId:     code.OpId,
}

// ProcessLoadConstInstr compiles a LoadConst instruction.
func (ic instrCompiler) ProcessLoadConstInstr(l ir.LoadConst) {
	k := ic.GetConstant(l.Kidx)
	dst := ic.codeReg(l.Dst)
	var opcode code.Opcode
	var inlined bool
	// Short strings and small integers are inlined.
	switch kk := k.(type) {
	case ir.Int:
		opcode, inlined = code.LoadSmallInt(dst, int64(kk))
	case ir.String:
		opcode, inlined = code.LoadShortString(dst, []byte(kk))
	case ir.Bool:
		opcode, inlined = code.LoadBool(dst, bool(kk)), true
	case ir.NilType:
		opcode, inlined = code.LoadNil(dst), true
	}
	if !inlined {
		ckidx := ic.QueueConstant(l.Kidx)
		opcode = code.LoadConst(dst, code.KIndexFromInt(ckidx))
	}
	ic.Emit(opcode)
}

// ProcessPushInstr compiles a Push instruction.
func (ic instrCompiler) ProcessPushInstr(p ir.Push) {
	var opcode code.Opcode
	if p.Etc {
		opcode = code.PushEtc(ic.codeReg(p.Cont), ic.codeReg(p.Item))
	} else {
		opcode = code.Push(ic.codeReg(p.Cont), ic.codeReg(p.Item))
	}
	ic.Emit(opcode)
}

// ProcessJumpInstr compiles a Jump instruction.
func (ic instrCompiler) ProcessJumpInstr(j ir.Jump) {
	// The offset will be computed later on - we don't know it at this stage.
	opcode := code.Jump(0)
	ic.EmitJump(opcode, code.Label(j.Label))
}

// ProcessJumpIfCmpInstr compiles a JumpIfCmp instruction: the comparison,
// then the jump it takes or steps over.
func (ic instrCompiler) ProcessJumpIfCmpInstr(j ir.JumpIfCmp) {
	codeOp, ok := codeBinOp[j.Op]
	if !ok {
		panic(fmt.Sprintf("Cannot compile %v: invalid op", j))
	}
	ic.Emit(code.CompareJump(codeOp, ic.codeReg(j.Lsrc), ic.codeReg(j.Rsrc), j.Not))
	ic.EmitJump(code.Jump(0), code.Label(j.Label))
}

// ProcessJumpIfInstr compiles a JumpIf instruction.
func (ic instrCompiler) ProcessJumpIfInstr(j ir.JumpIf) {
	var opcode code.Opcode
	if j.Not {
		opcode = code.JumpIfNot(0, ic.codeReg(j.Cond))
	} else {
		opcode = code.JumpIf(0, ic.codeReg(j.Cond))
	}
	ic.EmitJump(opcode, code.Label(j.Label))
}

// ProcessCallInstr compiles a Call instruction.
func (ic instrCompiler) ProcessCallInstr(c ir.Call) {
	call := code.CallArgs
	if c.Tail {
		call = code.TailCallArgs
	}
	ic.Emit(call(ic.codeReg(c.Cont), len(c.Args)))
	args := make([]code.Reg, len(c.Args))
	clear := make([]bool, len(c.Args))
	for i, r := range c.Args {
		args[i] = ic.codeReg(r)
		// A temporary that is not taken at this point is read for the last
		// time by this call (computeLifetimes releases a register just
		// before its last read), so nothing reads it after: the call clears
		// it once pushed, so that its value does not stay reachable from the
		// frame.  A register listed again later in the same call is cleared
		// at its last listing.  A taken register is read later; a named
		// local keeps its value.
		clear[i] = ic.regs[args[i].Idx()] == 0 &&
			!ic.registers[r].IsLocal && !slices.Contains(c.Args[i+1:], r)
	}
	for _, w := range code.ArgWords(args, clear) {
		ic.Emit(w)
	}
}

// ProcessMkClosureInstr compiles a MkClosure instruction.
func (ic instrCompiler) ProcessMkClosureInstr(m ir.MkClosure) {
	ckidx := ic.QueueConstant(m.Code)
	opcode := code.LoadClosure(ic.codeReg(m.Dst), code.KIndexFromInt(ckidx))
	ic.Emit(opcode)
	// Now add the upvalues: the boxes the closure captures.
	for _, upval := range m.Upvalues {
		ic.Emit(code.Upval(ic.codeReg(m.Dst), ic.codeReg(upval)))
	}
}

// ProcessMkContInstr compiles a MkCont instruction.
func (ic instrCompiler) ProcessMkContInstr(m ir.MkCont) {
	var opcode code.Opcode
	if m.Tail {
		opcode = code.TailCont(ic.codeReg(m.Dst), ic.codeReg(m.Closure))
	} else {
		opcode = code.Cont(ic.codeReg(m.Dst), ic.codeReg(m.Closure))
	}
	ic.Emit(opcode)
}

// ProcessClearRegInstr compiles a ClearReg instruction.  After the lowering
// of cells (ir.LowerCells) a clear concerns a plain register, which it
// resets to nil.
func (ic instrCompiler) ProcessClearRegInstr(i ir.ClearReg) {
	ic.Emit(code.LoadNil(ic.codeReg(i.Dst)))
}

// ProcessMkTableInstr compiles a MkTable instruction.
func (ic instrCompiler) ProcessMkTableInstr(m ir.MkTable) {
	opcode := code.LoadTable(ic.codeReg(m.Dst), m.NValues, m.NFields)
	ic.Emit(opcode)
}

// ProcessMkVarargTableInstr compiles a MkVarargTable instruction.
func (ic instrCompiler) ProcessMkVarargTableInstr(m ir.MkVarargTable) {
	ic.Emit(code.MkVarargTable(ic.codeReg(m.Dst), ic.codeReg(m.Etc)))
}

// ProcessLookupInstr compiles a Lookup instruction.
func (ic instrCompiler) ProcessLookupInstr(s ir.Lookup) {
	opcode := code.LoadLookup(ic.codeReg(s.Dst), ic.codeReg(s.Table), ic.codeReg(s.Index))
	ic.Emit(opcode)
}

// ProcessLookupKInstr compiles a LookupK instruction.  The constant index
// must fit in 8 bits for the constant key opcode; otherwise the key is loaded
// into a register first, as it would be for a Lookup.  That register is the
// destination unless it also holds the table, in which case a scratch
// register is used.
func (ic instrCompiler) ProcessLookupKInstr(s ir.LookupK) {
	dst := ic.codeReg(s.Dst)
	tbl := ic.codeReg(s.Table)
	ckidx := ic.QueueConstant(s.Kidx)
	if ckidx <= math.MaxUint8 {
		ic.Emit(code.LoadLookupK(dst, tbl, code.KIndex8FromInt(ckidx)))
		return
	}
	key := dst
	if key == tbl {
		key = ic.scratchReg(tbl)
	}
	ic.Emit(code.LoadConst(key, code.KIndexFromInt(ckidx)))
	ic.Emit(code.LoadLookup(dst, tbl, key))
}

// ProcessSetIndexKInstr compiles a SetIndexK instruction.  The constant index
// must fit in 8 bits for the constant key opcode; otherwise the key is loaded
// into a scratch register first, as it would be for a SetIndex.
func (ic instrCompiler) ProcessSetIndexKInstr(s ir.SetIndexK) {
	src := ic.codeReg(s.Src)
	tbl := ic.codeReg(s.Table)
	ckidx := ic.QueueConstant(s.Kidx)
	if ckidx <= math.MaxUint8 {
		ic.Emit(code.SetIndexK(src, tbl, code.KIndex8FromInt(ckidx)))
		return
	}
	key := ic.scratchReg(src, tbl)
	ic.Emit(code.LoadConst(key, code.KIndexFromInt(ckidx)))
	ic.Emit(code.SetIndex(src, tbl, key))
}

// ProcessSetIndexInstr compiles a SetIndex instruction.
func (ic instrCompiler) ProcessSetIndexInstr(s ir.SetIndex) {
	tReg := ic.codeReg(s.Table)
	iReg := ic.codeReg(s.Index)
	if s.CheckNotDefined {
		ic.Emit(code.CheckNotDefined(tReg, iReg))
	}
	ic.Emit(code.SetIndex(ic.codeReg(s.Src), tReg, iReg))
}

// ProcessReceiveInstr compiles a Receive instruction.
func (ic instrCompiler) ProcessReceiveInstr(r ir.Receive) {
	entry := ic.atEntry // receiving the parameters is still the entry
	for _, reg := range r.Dst {
		ic.Emit(code.Receive(ic.codeReg(reg)))
	}
	ic.atEntry = entry
}

// ProcessReceiveEtcInstr compiles a ReceiveEtc instruction.
func (ic instrCompiler) ProcessReceiveEtcInstr(r ir.ReceiveEtc) {
	entry := ic.atEntry
	for _, reg := range r.Dst {
		ic.Emit(code.Receive(ic.codeReg(reg)))
	}
	ic.Emit(code.ReceiveEtc(ic.codeReg(r.Etc)))
	ic.atEntry = entry
}

// ProcessEtcLookupInstr compiles a EtcLookup instruction.
func (ic instrCompiler) ProcessEtcLookupInstr(l ir.EtcLookup) {
	if l.Idx < 0 || l.Idx >= 256 {
		panic("Etc lookup index out of range")
	}
	ic.Emit(code.LoadEtcLookup(ic.codeReg(l.Dst), ic.codeReg(l.Etc), l.Idx))
}

// ProcessFillTableInstr compiles a FillTable instruction.
func (ic instrCompiler) ProcessFillTableInstr(f ir.FillTable) {
	if f.Idx < 0 || f.Idx >= 256 {
		panic("Fill table index out of range")
	}
	ic.Emit(code.FillTable(ic.codeReg(f.Dst), ic.codeReg(f.Etc), f.Idx))
}

// ProcessTruncateCloseStackInstr compiles a TruncateCloseStack instruction.
func (ic instrCompiler) ProcessTruncateCloseStackInstr(t ir.TruncateCloseStack) {
	if t.Height < 0 || t.Height >= 65536 {
		panic("close stack height out of range")
	}
	ic.Emit(code.ClTrunc(uint16(t.Height)))
}

// ProcessPushCloseStackInstr compiles a PushCloseStack instruction.
func (ic instrCompiler) ProcessPushCloseStackInstr(p ir.PushCloseStack) {
	ic.Emit(code.ClPush(ic.codeReg(p.Src)))
}

// ProcessPrepForLoopInstr compiles a PrepForLoop instruction: the opcode and
// the jump it takes when the loop has no iteration.
func (ic instrCompiler) ProcessPrepForLoopInstr(i ir.PrepForLoop) {
	ic.Emit(code.PrepForLoop(ic.codeReg(i.Start), ic.codeReg(i.Stop), ic.codeReg(i.Step)))
	ic.EmitJump(code.Jump(0), code.Label(i.Label))
}

// ProcessAdvForLoopInstr compiles an AdvForLoop instruction: the opcode and
// the jump it takes when the loop has another iteration.
func (ic instrCompiler) ProcessAdvForLoopInstr(i ir.AdvForLoop) {
	ic.Emit(code.AdvForLoop(ic.codeReg(i.Start), ic.codeReg(i.Stop), ic.codeReg(i.Step)))
	ic.EmitJump(code.Jump(0), code.Label(i.Label))
}

func (ic instrCompiler) ProcessTakeRegisterInstr(t ir.TakeRegister) {
	ic.takeRegister(t.Reg)
}

func (ic instrCompiler) ProcessReleaseRegisterInstr(r ir.ReleaseRegister) {
	ic.releaseRegister(r.Reg)
}

func (ic instrCompiler) ProcessDeclareLabelInstr(l ir.DeclareLabel) {
	ic.builder.EmitLabel(code.Label(l.Label))
}

type regAllocation struct {
	r    code.Reg
	done bool
}
type regAllocator struct {
	registers   []ir.RegData
	allocations []regAllocation
	regs        []int
	upvalueOf   map[ir.Register]int // the upvalue a register receives, if any

	// atEntry is true until an opcode other than the parameters' recv has
	// been emitted; entryLoads are the loads of fixed upvalues found there,
	// which the frame does instead of an opcode (code.EntryLoad).  boxUsed
	// marks the registers some opcode uses as a box (boxUses); the frame
	// must put the upvalues among them in their registers.
	atEntry    bool
	entryLoads []code.EntryLoad
	boxUsed    map[ir.Register]bool
}

func (a *regAllocator) takeRegister(r ir.Register) int {
	cr := a.codeReg(r)
	a.regs[cr.Idx()]++
	return a.regs[cr.Idx()]
}

func (a *regAllocator) releaseRegister(r ir.Register) int {
	cr := a.codeReg(r)
	a.regs[cr.Idx()]--
	return a.regs[cr.Idx()]
}

func (a *regAllocator) codeReg(r ir.Register) code.Reg {
	alloc := a.allocations[r]
	if alloc.done {
		return alloc.r
	}
	var i uint8
	a.regs, i = allocReg(a.regs)
	cr := code.ValueReg(i)
	a.allocations[r] = regAllocation{
		done: true,
		r:    cr,
	}
	return cr
}

// scratchReg returns a value register that is not taken and is none of the
// given registers (the operands of the current instruction, which may hold
// untaken temporaries).  It stays free, so it can only be used until the next
// register is allocated.
func (a *regAllocator) scratchReg(avoid ...code.Reg) code.Reg {
	for i, c := range a.regs {
		r := code.ValueReg(uint8(i))
		if c != 0 {
			continue
		}
		free := true
		for _, x := range avoid {
			free = free && x != r
		}
		if free {
			return r
		}
	}
	if len(a.regs) == math.MaxUint8 {
		panic(notEnoughRegisters())
	}
	a.regs = append(a.regs, 0)
	return code.ValueReg(uint8(len(a.regs) - 1))
}

func allocReg(regs []int) ([]int, uint8) {
	for i, c := range regs {
		if c == 0 {
			return regs, uint8(i)
		}
	}
	if len(regs) == math.MaxUint8 {
		panic(notEnoughRegisters())
	}
	i := len(regs)
	return append(regs, 0), uint8(i)
}

// ErrNotEnoughRegisters is the cause of a compilation error when a function
// needs more registers than an opcode can name.
var ErrNotEnoughRegisters = errors.New("not enough registers")

type CompilationPanic struct {
	msg  string
	line int32
	err  error
}

// Unwrap returns the cause of the error, if there is one.
func (p *CompilationPanic) Unwrap() error {
	return p.err
}

func (p *CompilationPanic) Error() string {
	if p.line > 0 {
		return fmt.Sprintf("%s (around line %d)", p.msg, p.line)
	}
	return p.msg
}

func notEnoughRegisters() *CompilationPanic {
	return &CompilationPanic{msg: ErrNotEnoughRegisters.Error(), err: ErrNotEnoughRegisters}
}
