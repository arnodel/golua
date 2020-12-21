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
	"github.com/arnodel/golua/lib/iolib"
	rt "github.com/arnodel/golua/runtime"
)

type luaCmd struct {
	disFlag        bool
	astFlag        bool
	unbufferedFlag bool
}

func (c *luaCmd) setFlags() {
	flag.BoolVar(&c.disFlag, "dis", false, "Disassemble source instead of running it")
	flag.BoolVar(&c.astFlag, "ast", false, "Print AST instead of running code")
	flag.BoolVar(&c.unbufferedFlag, "u", false, "Force unbuffered output")
}

func (c *luaCmd) run() int {
	var (
		chunkName string
		chunk     []byte
		err       error
		args      []string
	)

	buffered := !isaTTY(os.Stdin) || flag.NArg() > 0
	if c.unbufferedFlag {
		buffered = false
	}
	iolib.BufferedStdFiles = buffered

	// Get a Lua runtime
	r := rt.New(nil)
	cleanup := lib.LoadAll(r)
	defer cleanup()

	// Run finalizers before we exit
	defer runtime.GC()

	switch flag.NArg() {
	case 0:
		chunkName = "<stdin>"
		if isaTTY(os.Stdin) {
			return repl(r)
		}
		chunk, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			fatal("Error reading <stdin>: %s", err)
		}
	default:
		chunkName = flag.Arg(0)
		chunk, err = ioutil.ReadFile(chunkName)
		if err != nil {
			return fatal("Error reading '%s': %s", chunkName, err)
		}
		args = flag.Args()[1:]
	}
	if c.astFlag {
		stat, err := rt.ParseLuaChunk(chunkName, chunk)
		if err != nil {
			return fatal("Error parsing %s: %s", chunkName, err)
		}
		w := ast.NewIndentWriter(os.Stdout)
		stat.HWrite(w)
		return 0
	}
	chunk = removeSlashBangLine(chunk)
	unit, err := rt.CompileLuaChunk(chunkName, chunk)
	if err != nil {
		return fatal("Error parsing %s: %s", chunkName, err)
	}

	if c.disFlag {
		unit.Disassemble(os.Stdout)
		return 0
	}

	var argVals []rt.Value
	if len(args) > 0 {
		argTable := rt.NewTable()
		argVals = make([]rt.Value, len(args))
		for i, arg := range args {
			argVal := rt.StringValue(arg)
			argTable.Set(rt.IntValue(int64(i+1)), argVal)
			argVals[i] = argVal
		}
		r.GlobalEnv().Set(rt.StringValue("arg"), rt.TableValue(argTable))
	}

	clos := rt.LoadLuaUnit(unit, r.GlobalEnv())
	cerr := rt.Call(r.MainThread(), rt.FunctionValue(clos), argVals, rt.NewTerminationWith(0, false))
	if cerr != nil {
		return fatal("!!! %s", cerr.Traceback())
	}
	return 0
}

func fatal(tpl string, args ...interface{}) int {
	fmt.Fprintf(os.Stderr, tpl, args...)
	return 1
}

func isaTTY(f *os.File) bool {
	fi, _ := f.Stat()
	return fi.Mode()&os.ModeCharDevice != 0
}

func removeSlashBangLine(chunk []byte) []byte {
	if len(chunk) < 2 {
		return chunk
	}
	if chunk[0] != '#' || chunk[1] != '!' {
		return chunk
	}
	i := 3
	for i < len(chunk) {
		if chunk[i] == '\n' || chunk[i] == '\r' {
			return chunk[i+1:]
		}
	}
	return nil
}

func repl(r *rt.Runtime) int {
	reader := bufio.NewReader(os.Stdin)
	w := new(bytes.Buffer)
	for {
		if len(w.Bytes()) == 0 {
			fmt.Print("> ")
		} else {
			fmt.Print("| ")
		}
		line, err := reader.ReadBytes('\n')
		line = bytes.TrimLeft(line, ">|")
		if err == io.EOF {
			w.WriteTo(os.Stdout)
			fmt.Print(string(line))
			return 0
		}
		_, err = w.Write(line)
		if err != nil {
			return fatal("error: %s", err)
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
		return snErr.IsUnexpectedEOF(), err
	}
	t := r.MainThread()
	term := rt.NewTerminationWith(0, true)
	cerr := rt.Call(t, rt.FunctionValue(clos), nil, term)
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
