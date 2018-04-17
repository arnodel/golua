package ast

import (
	"strconv"
	"strings"

	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/token"
)

func NumberFromString(nstring string) (ExpNode, error) {
	if strings.ContainsAny(nstring, ".eE") {
		f, err := strconv.ParseFloat(nstring, 64)
		if err != nil {
			return nil, err
		}
		return Float(f), nil
	}
	n, err := strconv.ParseInt(nstring, 0, 64)
	if err != nil {
		return nil, err
	}
	return Int(n), nil
}

func NewNumber(id *token.Token) (ExpNode, error) {
	return NumberFromString(string(id.Lit))
}

type Int int64

func (n Int) HWrite(w HWriter) {
	w.Writef("%d", int64(n))
}

func (n Int) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	ir.EmitConstant(c, ir.Int(n), dst)
	return dst
}

type Float float64

func (f Float) CompileExp(c *ir.Compiler, dst ir.Register) ir.Register {
	ir.EmitConstant(c, ir.Float(f), dst)
	return dst
}

func (f Float) HWrite(w HWriter) {
	w.Writef("%f", float64(f))
}
