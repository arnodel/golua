package ir

import (
	"fmt"
	"strings"

	"github.com/arnodel/golua/ops"
)

type Instruction interface {
	fmt.Stringer
}

type Register int

func (r Register) String() string {
	if r >= 0 {
		return fmt.Sprintf("r%d", r)
	}
	return fmt.Sprintf("u%d", -r)
}

type Label uint

func (l Label) String() string {
	return fmt.Sprintf("L%d", l)
}

type Combine struct {
	Op   ops.Op
	Dst  Register
	Lsrc Register
	Rsrc Register
}

func (c Combine) String() string {
	return fmt.Sprintf("%s := %s(%s, %s)", c.Dst, c.Op, c.Lsrc, c.Rsrc)
}

type Transform struct {
	Op  ops.Op
	Dst Register
	Src Register
}

func (t Transform) String() string {
	return fmt.Sprintf("%s := %s(%s)", t.Dst, t.Op, t.Src)
}

type LoadConst struct {
	Dst  Register
	Kidx uint
}

func (l LoadConst) String() string {
	return fmt.Sprintf("%s := k%d", l.Dst, l.Kidx)
}

type Push struct {
	Cont Register
	Item Register
}

func (p Push) String() string {
	return fmt.Sprintf("push %s to %s", p.Item, p.Cont)
}

type PushCC struct {
	Cont Register
}

func (p PushCC) String() string {
	return fmt.Sprintf("push cc to %s", p.Cont)
}

type Jump struct {
	Label Label
}

func (j Jump) String() string {
	return fmt.Sprintf("jump %s", j.Label)
}

type JumpIf struct {
	Cond  Register
	Label Label
	Not   bool
}

func (j JumpIf) String() string {
	return fmt.Sprintf("jump %s if %s is not %t", j.Label, j.Cond, j.Not)
}

type Call struct {
	Cont Register
}

func (c Call) String() string {
	return fmt.Sprintf("call %s", c.Cont)
}

type MkClosure struct {
	Dst      Register
	Code     uint
	Upvalues []Register
}

func joinRegisters(regs []Register, sep string) string {
	us := []string{}
	for _, r := range regs {
		us = append(us, r.String())
	}
	return strings.Join(us, sep)
}

func (m MkClosure) String() string {
	return fmt.Sprintf("%s := mkclos(k%d; %s)", m.Dst, m.Code, joinRegisters(m.Upvalues, ", "))
}

type MkCont struct {
	Dst     Register
	Closure Register
}

func (m MkCont) String() string {
	return fmt.Sprintf("%s := mkcont(%s)", m.Dst, m.Closure)
}

type MkCell struct {
	Dst Register
	Src Register
}

func (m MkCell) String() string {
	return fmt.Sprintf("%s := mkcell(%s)", m.Dst, m.Src)
}

type MkTable struct {
	Dst Register
}

func (m MkTable) String() string {
	return fmt.Sprintf("%s := mktable()", m.Dst)
}

type Lookup struct {
	Dst   Register
	Table Register
	Index Register
}

func (s Lookup) String() string {
	return fmt.Sprintf("%s := %s[%s]", s.Dst, s.Table, s.Index)
}

type SetIndex struct {
	Table Register
	Index Register
	Src   Register
}

func (s SetIndex) String() string {
	return fmt.Sprintf("%s[%s] := %s", s.Table, s.Index, s.Src)
}

// type Receiver interface {
// 	GetRegisters() []Register
// 	HasEtc() bool
// 	GetEtc() Register
// }

type Receive struct {
	Dst []Register
}

// func (r Receive) GetRegisters() []Register { return r.Dst }
// func (r Receive) HasEtc() bool             { return false }
// func (r Receive) GetEtc() Register         { return Register(0) }

func (r Receive) String() string {
	return fmt.Sprintf("recv(%s)", joinRegisters(r.Dst, ", "))
}

type ReceiveEtc struct {
	Dst []Register
	Etc Register
}

// func (r ReceiveEtc) GetRegisters() []Register { return r.Dst }
// func (r ReceiveEtc) HasEtc() bool             { return true }
// func (r ReceiveEtc) GetEtc() Register         { return r.Etc }

func (r ReceiveEtc) String() string {
	return fmt.Sprintf("recv(%s, ...%s)", joinRegisters(r.Dst, ", "), r.Etc)
}

type JumpIfForLoopDone struct {
	Label Label
	Var   Register
	Limit Register
	Step  Register
}

func (j JumpIfForLoopDone) String() string {
	return fmt.Sprintf("jump %s if for loop done(%s, %s, %s)", j.Label, j.Var, j.Limit, j.Step)
}
