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

	ccerrors "github.com/arnodel/golua/errors"
	"github.com/arnodel/golua/lib/base"
	"github.com/arnodel/golua/lib/coroutine"
	"github.com/arnodel/golua/lib/packagelib"
	"github.com/arnodel/golua/lib/stringlib"
	"github.com/arnodel/golua/runtime"
	"github.com/arnodel/golua/token"
)

func main() {
	disFlag := flag.Bool("dis", false, "Disassemble source instead of running it")
	flag.Parse()
	var chunkName string
	var chunk []byte
	var err error

	r := runtime.New(os.Stdout)
	base.Load(r)
	coroutine.Load(r)
	packagelib.Load(r)
	stringlib.Load(r)

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
	unit, err := runtime.CompileLuaChunk(chunkName, chunk)
	if err != nil {
		fatal("Error parsing %s: %s", chunkName, err)
	}

	if *disFlag {
		unit.Disassemble(os.Stdout)
		return
	}

	clos := runtime.LoadLuaUnit(unit, r.GlobalEnv())
	cerr := runtime.Call(r.MainThread(), clos, nil, runtime.NewTerminationWith(0, false))
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

func repl(rt *runtime.Runtime) {
	r := bufio.NewReader(os.Stdin)
	w := new(bytes.Buffer)
	for {
		if len(w.Bytes()) == 0 {
			fmt.Print("> ")
		} else {
			fmt.Print("| ")
		}
		line, err := r.ReadBytes('\n')
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
		more, err := runChunk(rt, append([]byte("return "), w.Bytes()...))
		if err != nil {
			more, err = runChunk(rt, w.Bytes())
		}
		if !more {
			w = new(bytes.Buffer)
			if err != nil {
				fmt.Printf("!!! %s\n", err)
			}
		}
	}
}

func runChunk(r *runtime.Runtime, source []byte) (bool, error) {
	clos, err := runtime.CompileAndLoadLuaChunk("<stdin>", source, r.GlobalEnv())
	if err != nil {
		pErr, ok := err.(*ccerrors.Error)
		if !ok {
			return false, err
		}
		return pErr.ErrorToken.Type == token.EOF, err
	}
	t := r.MainThread()
	term := runtime.NewTerminationWith(0, true)
	cerr := runtime.Call(t, clos, nil, term)
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
