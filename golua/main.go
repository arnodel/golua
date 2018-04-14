package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/code"
	"github.com/arnodel/golua/lexer"
	"github.com/arnodel/golua/parser"
)

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Got %d args", flag.NArg())
		os.Exit(1)
	}
	path := flag.Arg(0)
	src, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	w := ast.NewIndentWriter(os.Stdout)
	p := parser.NewParser()
	s := lexer.NewLexer(src)
	tree, err := p.Parse(s)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("--- AST ---")
	tree.(ast.Node).HWrite(w)
	w.Next()
	fmt.Println("--- CODE ---")
	c := tree.(ast.BlockStat).CompileChunk()
	kc := c.NewConstantCompiler()
	unit := kc.CompileQueue()
	dis := code.NewUnitDisassembler(unit)
	dis.Disassemble(os.Stdout)
}
