package ast

import (
	"bytes"
	"strconv"
	"strings"

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
		return Float{Location: loc, Val: f}, nil
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
	return Int{Location: loc, Val: n}, nil
}

type Int struct {
	Location
	Val uint64
}

var _ ExpNode = Int{}

func NewInt(val uint64) Int {
	return Int{Val: val}
}

// HWrite prints a tree representation of the node.
func (n Int) HWrite(w HWriter) {
	w.Writef("%d", n.Val)
}

// ProcessExp uses the given ExpProcessor to process the receiver.
func (n Int) ProcessExp(p ExpProcessor) {
	p.ProcessIntExp(n)
}

type Float struct {
	Location
	Val float64
}

var _ ExpNode = Float{}

func NewFloat(x float64) Float {
	return Float{Val: x}
}

// ProcessExp uses the given ExpProcessor to process the receiver.
func (f Float) ProcessExp(p ExpProcessor) {
	p.ProcessFloatExp(f)
}

// HWrite prints a tree representation of the node.
func (f Float) HWrite(w HWriter) {
	w.Writef("%f", f.Val)
}

type NumberOracle interface {
	IsNumber() bool
}

func IsNumber(e ExpNode) bool {
	n, ok := e.(NumberOracle)
	return ok && n.IsNumber()
}
