package luatests

import (
	"bytes"
	"fmt"
	"io"

	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/lexer"
	"github.com/arnodel/golua/lib/base"
	"github.com/arnodel/golua/lib/coroutine"
	"github.com/arnodel/golua/parser"
	"github.com/arnodel/golua/runtime"
)

func RunSource(source []byte, output io.Writer) {
	p := parser.NewParser()
	s := lexer.NewLexer(source)
	tree, err := p.Parse(s)
	if err != nil {
		fmt.Fprintf(output, "!!! parse: %s", err)
		return
	}
	c := tree.(ast.BlockStat).CompileChunk()
	kc := c.NewConstantCompiler()
	unit := kc.CompileQueue()
	r := runtime.New(output)
	base.Load(r)
	coroutine.Load(r)
	t := r.MainThread()
	clos := runtime.LoadLuaUnit(t, unit)
	err = runtime.Call(t, clos, nil, runtime.NewTerminationWith(0, false))
	if err != nil {
		fmt.Fprintf(output, "!!! runtime: %s", err)
	}
}

func RunLuaTest(source []byte) error {
	outputBuf := new(bytes.Buffer)
	checkers := ExtractLineCheckers(source)
	RunSource(source, outputBuf)
	return CheckLines(outputBuf.Bytes(), checkers)
}
