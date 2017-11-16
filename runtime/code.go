package runtime

import (
	"errors"
	"fmt"

	"github.com/arnodel/golua/ops"
)

type Value interface{}

type Register uint16

func (r Register) GetType() uint8 {
	return uint8(r >> 8)
}

func (r Register) GetIndex() uint8 {
	return uint8(r & 0xFF)
}

type Code []uint32

func (c *Code) DecodeAt(pc uint) {
	instr := (*c)[pc]
	switch instr & 0xF {
	}
}

type Closure struct {
	code          Code
	constants     []Value
	upvalues      []Value
	registerCount uint
}

func (c *Closure) NewContinuation() *Continuation {
	return &Continuation{
		code:      c.code,
		constants: c.constants,
		upvalues:  c.upvalues,
		registers: make([]Value, c.registerCount),
		pc:        0,
	}
}

type Continuation struct {
	code      Code
	constants []Value
	upvalues  []Value
	registers []Value
	pc        uint
	status    uint
}

const (
	Calling = iota
	Running
)

func (c *Continuation) Push(v Value) {
	// TODO
}

func (c *Continuation) Resume() {
}

func (c *Continuation) RegValue(reg Register) Value {
	if reg.GetType() == 0 {
		return c.registers[reg.GetIndex()]
	} else {
		return c.upvalues[reg.GetIndex()]
	}
}

func (c *Continuation) SetReg(reg Register, v Value) {
	if reg.GetType() == 0 {
		c.registers[reg.GetIndex()] = v
	} else {
		c.upvalues[reg.GetIndex()] = v
	}
}

func (c *Continuation) DoCombine(op ops.Op, dst Register, lsrc Register, rsrc Register) error {
	var res Value
	var err error
	switch op {
	case ops.OpAdd:
		res, err = add(c.RegValue(lsrc), c.RegValue(rsrc))
	default:
		return fmt.Errorf("Operator %s not implemented", op)
	}
	if err == nil {
		c.SetReg(dst, res)
	}
	c.pc++
	return err
}

func (c *Continuation) DoTransform(op ops.Op, dst Register, src Register) error {
	return fmt.Errorf("Transform not implemented")
}

func (c *Continuation) DoLoadConst(dst Register, kidx uint) error {
	c.SetReg(dst, c.constants[kidx])
	c.pc++
	return nil
}

func (c *Continuation) DoPush(dst Register, src Register) error {
	cont, ok := c.RegValue(dst).(*Continuation)
	if !ok {
		return errors.New("Push dst must be a continuation")
	}
	cont.Push(c.RegValue(src))
	c.pc++
	return nil
}

func (c *Continuation) DoPushCC(dst Register) error {
	cont, ok := c.RegValue(dst).(*Continuation)
	if !ok {
		return errors.New("PushCC dst must be a continuation")
	}
	cont.Push(c)
	c.pc++
	return nil
}

func (c *Continuation) DoJump(pos uint) error {
	c.pc = pos
	return nil
}

func (c *Continuation) DoJumpIf(pos uint, cond Register, not bool) error {
	v := TruthValue(c.RegValue(cond))
	if v != not {
		c.pc = pos
	}
	c.pc++
	return nil
}

func (c *Continuation) DoCall(reg Register) error {
	cont, ok := c.RegValue(reg).(*Continuation)
	if !ok {
		return errors.New("Call reg must be a continuation")
	}
	c.pc++
	c.status = Calling
	return nil
}

type Thread struct {
	currCont          *Continuation
	errCont           *Continuation
	resumeCh, yieldCh chan []Value
}

func (t *Thread) Resume(arg []Value) []Value {
	t.resumeCh <- arg
	return <-t.yieldCh

}

func (t *Thread) Yield(arg []Value) []Value {
	t.yieldCh <- arg
	return <-t.resumeCh
}

func (t *Thread) Run(s *Scheduler) {
	t.currCont.Resume()
}

type Scheduler struct {
	scheduledThreads chan *Thread
}

func (s *Scheduler) Schedule(th *Thread) {
	s.scheduledThreads <- th
}

func (s *Scheduler) Run() {
	for th := range s.scheduledThreads {
		th.Run(s)
	}
}
