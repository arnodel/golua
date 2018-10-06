package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"

	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/lib"
	"github.com/arnodel/golua/lib/base"
	rt "github.com/arnodel/golua/runtime"
)

func main() {
	disFlag := flag.Bool("dis", false, "Disassemble source instead of running it")
	astFlag := flag.Bool("ast", false, "Print AST instead of running code")
	flag.Parse()
	var chunkName string
	var chunk []byte
	var err error

	r := rt.New(os.Stdout)
	lib.Load(r)

	// Run finalizers before we exit
	defer runtime.GC()

	switch flag.NArg() {
	case 0:
		chunkName = "<stdin>"
		if isaTTY(os.Stdin) {
			repl(r)
			return
		}
		chunk, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			fatal("Error reading <stdin>: %s", err)
		}
	case 1:
		chunkName = flag.Arg(0)
		chunk, err = ioutil.ReadFile(chunkName)
		if err != nil {
			fatal("Error reading '%s': %s", chunkName, err)
		}
	default:
		fatal("At most 1 argument (lua file name)")
	}
	if *astFlag {
		stat, err := rt.ParseLuaChunk(chunkName, chunk)
		if err != nil {
			fatal("Error parsing %s: %s", chunkName, err)
		}
		w := ast.NewIndentWriter(os.Stdout)
		stat.HWrite(w)
		return
	}
	unit, err := rt.CompileLuaChunk(chunkName, chunk)
	if err != nil {
		fatal("Error parsing %s: %s", chunkName, err)
	}

	if *disFlag {
		unit.Disassemble(os.Stdout)
		return
	}

	clos := rt.LoadLuaUnit(unit, r.GlobalEnv())
	cerr := rt.Call(r.MainThread(), clos, nil, rt.NewTerminationWith(0, false))
	if cerr != nil {
		fatal("!!! %s", cerr.Traceback())
	}
}

func fatal(tpl string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, tpl, args...)
	os.Exit(1)
}

func isaTTY(f *os.File) bool {
	fi, _ := f.Stat()
	return fi.Mode()&os.ModeCharDevice != 0
}

func repl(r *rt.Runtime) {
	reader := bufio.NewReader(os.Stdin)
	w := new(bytes.Buffer)
	for {
		if len(w.Bytes()) == 0 {
			fmt.Print("> ")
		} else {
			fmt.Print("| ")
		}
		line, err := reader.ReadBytes('\n')
		if err == io.EOF {
			w.WriteTo(os.Stdout)
			fmt.Print(string(line))
			return
		}
		_, err = w.Write(line)
		if err != nil {
			return
		}
		// This is a trick to be able to evaluate lua expressions in the repl
		more, err := runChunk(r, append([]byte("return "), w.Bytes()...))
		if err != nil {
			more, err = runChunk(r, w.Bytes())
		}
		if !more {
			w = new(bytes.Buffer)
			if err != nil {
				fmt.Printf("!!! %s\n", err)
			}
		}
	}
}

func runChunk(r *rt.Runtime, source []byte) (bool, error) {
	clos, err := rt.CompileAndLoadLuaChunk("<stdin>", source, r.GlobalEnv())
	if err != nil {
		snErr, ok := err.(*rt.SyntaxError)
		if !ok {
			return false, err
		}
		return snErr.Type == rt.ErrSyntaxEOF, err
	}
	t := r.MainThread()
	term := rt.NewTerminationWith(0, true)
	cerr := rt.Call(t, clos, nil, term)
	if cerr == nil {
		if len(term.Etc()) > 0 {
			cerr = base.Print(t, term.Etc())
			if cerr != nil {
				return false, errors.New(cerr.Traceback())
			}
		}
		return false, nil
	}
	return false, errors.New(cerr.Traceback())
}
