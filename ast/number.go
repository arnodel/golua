package ast

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/token"
)

func isFloatToken(tok *token.Token) bool {
	switch tok.Type {
	case token.NUMDEC:
		return bytes.ContainsAny(tok.Lit, ".eE")
	case token.NUMHEX:
		return bytes.ContainsAny(tok.Lit, ".pP")
	default:
		return false
	}
}

func NewNumber(id *token.Token) (ExpNode, error) {
	loc := LocFromToken(id)
	nstring := string(id.Lit)
	if isFloatToken(id) {
		f, err := strconv.ParseFloat(nstring, 64)
		if err != nil {
			return nil, err
		}
		return Float{Location: loc, val: f}, nil
	}
	var n uint64
	var err error
	if strings.HasPrefix(nstring, "0x") || strings.HasPrefix(nstring, "0X") {
		nstring = nstring[2:]
		if len(nstring) > 16 {
			// A hex integral constant is "truncated" if too long (more than 8 bytes)
			nstring = nstring[len(nstring)-16:]
		}
		n, err = strconv.ParseUint(nstring, 16, 64)
	} else {
		n, err = strconv.ParseUint(nstring, 10, 64)
	}
	if err != nil {
		return nil, err
	}
	return Int{Location: loc, val: n}, nil
}

type Int struct {
	Location
	val uint64
}

func NewInt(val uint64) Int {
	return Int{val: val}
}

func (n Int) Val() uint64 {
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
