package ast

import (
	"strconv"
	"strings"

	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/token"
)

func NewNumber(id *token.Token) (ExpNode, error) {
	loc := LocFromToken(id)
	nstring := string(id.Lit)
	if strings.ContainsAny(nstring, ".eE") {
		f, err := strconv.ParseFloat(nstring, 64)
		if err != nil {
			return nil, err
		}
		return Float{Location: loc, val: f}, nil
	}
	n, err := strconv.ParseInt(nstring, 0, 64)
	if err != nil {
		return nil, err
	}
	return Int{Location: loc, val: n}, nil
}

type Int struct {
	Location
	val int64
}

func NewInt(val int64) Int {
	return Int{val: val}
}

func (n Int) Val() int64 {
	return n.val
}

func (n Int) HWrite(w HWriter) {
	w.Writef("%d", n.val)
}

func (n Int) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	EmitLoadConst(c, n, ir.Int(n.val), dst)
	return dst
}

type Float struct {
	Location
	val float64
}

func (f Float) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	EmitLoadConst(c, f, ir.Float(f.val), dst)
	return dst
}

func (f Float) HWrite(w HWriter) {
	w.Writef("%f", f.val)
}

func (f Float) Val() float64 {
	return f.val
}
