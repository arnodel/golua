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
	"strings"

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
	cpuLimit       uint64
	memLimit       uint64
	flags          string
}

func (c *luaCmd) setFlags() {
	flag.BoolVar(&c.disFlag, "dis", false, "Disassemble source instead of running it")
	flag.BoolVar(&c.astFlag, "ast", false, "Print AST instead of running code")
	flag.BoolVar(&c.unbufferedFlag, "u", false, "Force unbuffered output")

	if rt.QuotasAvailable {
		flag.Uint64Var(&c.cpuLimit, "cpulimit", 0, "CPU limit")
		flag.Uint64Var(&c.memLimit, "memlimit", 0, "memory limit")
		flag.StringVar(&c.flags, "flags", "", "compliance flags turned on")
	}
}

func (c *luaCmd) run() (retcode int) {
	var (
		chunkName string
		chunk     []byte
		err       error
		args      []string
		flags     rt.ComplianceFlags
	)

	buffered := !isaTTY(os.Stdin) || flag.NArg() > 0
	if c.unbufferedFlag {
		buffered = false
	}
	iolib.BufferedStdFiles = buffered

	if c.flags != "" {
		for _, name := range strings.Split(c.flags, ",") {
			var ok bool
			flags, ok = flags.AddFlagWithName(name)
			if !ok {
				return fatal("Unknown flag: %s", name)
			}
		}
	}

	// Get a Lua runtime
	r := rt.New(nil)
	r.PushContext(rt.RuntimeContextDef{
		HardLimits: rt.RuntimeResources{
			Cpu: c.cpuLimit,
			Mem: c.memLimit,
		},
		SafetyFlags: flags,
	})
	cleanup := lib.LoadAll(r)
	defer cleanup()

	// Run finalizers before we exit
	defer runtime.GC()

	switch flag.NArg() {
	case 0:
		chunkName = "<stdin>"
		if isaTTY(os.Stdin) {
			return c.repl(r)
		}
		chunk, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			return fatal("Error reading <stdin>: %s", err)
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
		stat, _, err := r.ParseLuaChunk(chunkName, chunk)
		if err != nil {
			return fatal("Error parsing %s: %s", chunkName, err)
		}
		w := ast.NewIndentWriter(os.Stdout)
		stat.HWrite(w)
		return 0
	}
	chunk = removeSlashBangLine(chunk)
	unit, _, err := r.CompileLuaChunk(chunkName, chunk)
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
			r.SetTable(argTable, rt.IntValue(int64(i+1)), argVal)
			argVals[i] = argVal
		}
		r.SetTable(r.GlobalEnv(), rt.StringValue("arg"), rt.TableValue(argTable))
	}

	defer func() {
		if rec := recover(); rec != nil {
			quotaExceeded, ok := rec.(rt.ContextTerminationError)
			if !ok {
				panic(r)
			}
			fmt.Fprintf(os.Stderr, "%s\n", quotaExceeded)
			retcode = 2
		}
	}()
	clos := r.LoadLuaUnit(unit, r.GlobalEnv())
	cerr := rt.Call(r.MainThread(), rt.FunctionValue(clos), argVals, rt.NewTerminationWith(0, false))
	if cerr != nil {
		return fatal("!!! %s", cerr.Traceback())
	}
	return 0
}

func fatal(tpl string, args ...interface{}) int {
	fmt.Fprintf(os.Stderr, tpl+"\n", args...)
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

func (c *luaCmd) repl(r *rt.Runtime) int {
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
		more, err := c.runChunk(r, append([]byte("return "), w.Bytes()...))
		if err != nil {
			more, err = c.runChunk(r, w.Bytes())
		}
		if !more {
			w = new(bytes.Buffer)
			if err != nil {
				fmt.Printf("!!! %s\n", err)
				if _, ok := err.(rt.ContextTerminationError); ok {
					fmt.Print("Reset limits and continue? [yN] ")
					line, err := reader.ReadString('\n')
					if err == io.EOF || strings.TrimSpace(line) != "y" {
						return 0
					}
					r.ResetQuota()
				}
			}
		}
	}
}

func (c *luaCmd) runChunk(r *rt.Runtime, source []byte) (more bool, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			quotaExceeded, ok := rec.(rt.ContextTerminationError)
			if !ok {
				panic(r)
			}
			err = quotaExceeded
			more = false
		}
	}()
	clos, err := r.CompileAndLoadLuaChunk("<stdin>", source, r.GlobalEnv())
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
