package main

import (
	"bufio"
	"bytes"
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
	"github.com/arnodel/golua/lib/debuglib"
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
	exec           execFlags

	complianceFlags rt.ComplianceFlags
}

func (c *luaCmd) setFlags() {
	flag.BoolVar(&c.disFlag, "dis", false, "Disassemble source instead of running it")
	flag.BoolVar(&c.astFlag, "ast", false, "Print AST instead of running code")
	flag.BoolVar(&c.unbufferedFlag, "u", false, "Force unbuffered output")
	flag.Var(&c.exec, "e", "statement to execute")

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
		readStdin bool
		repl      bool
	)

	buffered := !isaTTY(os.Stdin) || flag.NArg() > 0
	if c.unbufferedFlag {
		buffered = false
	}
	iolib.BufferedStdFiles = buffered

	if c.flags != "" {
		for _, name := range strings.Split(c.flags, ",") {
			var ok bool
			c.complianceFlags, ok = c.complianceFlags.AddFlagWithName(name)
			if !ok {
				return fatal("Unknown flag: %s", name)
			}
		}
	}

	// Get a Lua runtime
	r := rt.New(nil)
	c.pushContext(r)

	cleanup := lib.LoadAll(r)
	defer cleanup()

	// Run finalizers before we exit
	defer runtime.GC()

	if len(c.exec) == 0 && flag.NArg() == 0 {
		chunkName = "<stdin>"
		readStdin = true
		repl = isaTTY(os.Stdin)
	}
	if flag.NArg() > 0 {
		chunkName = flag.Arg(0)
		chunk, err = ioutil.ReadFile(chunkName)
		if err != nil {
			return fatal("Error reading '%s': %s", chunkName, err)
		}
		args = flag.Args()[1:]
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

	for _, src := range c.exec {
		unit, _, err := r.CompileLuaChunk("<exec>", []byte(src))
		if err != nil {
			return fatal("Error parsing %q: %s", src, err)
		}
		clos := r.LoadLuaUnit(unit, rt.TableValue(r.GlobalEnv()))
		cerr := rt.Call(r.MainThread(), rt.FunctionValue(clos), argVals, rt.NewTerminationWith(nil, 0, false))
		if cerr != nil {
			return fatal("!!! %s", cerr.Error())
		}
	}

	if readStdin {
		if repl {
			return c.repl(r)
		}
		chunk, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			return fatal("Error reading <stdin>: %s", err)
		}
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

	if c.disFlag {
		unit, _, err := r.CompileLuaChunk(chunkName, chunk)
		if err != nil {
			return fatal("Error parsing %s: %s", chunkName, err)
		}
		unit.Disassemble(os.Stdout)
		return 0
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

	clos, err := r.LoadFromSourceOrCode(chunkName, chunk, "bt", rt.TableValue(r.GlobalEnv()), true)
	if err != nil {
		return fatal("Error loading %s: %s", chunkName, err)
	}
	cerr := rt.Call(r.MainThread(), rt.FunctionValue(clos), argVals, rt.NewTerminationWith(nil, 0, false))
	if cerr != nil {
		return fatal("!!! %s", cerr.Error())
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
					r.PopContext()
					c.pushContext(r)
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
	clos, err := r.CompileAndLoadLuaChunk("<stdin>", source, rt.TableValue(r.GlobalEnv()))
	if err != nil {
		snErr, ok := err.(*rt.SyntaxError)
		if !ok {
			return false, err
		}
		return snErr.IsUnexpectedEOF(), err
	}
	t := r.MainThread()
	term := rt.NewTerminationWith(nil, 0, true)
	cerr := rt.Call(t, rt.FunctionValue(clos), nil, term)
	if cerr == nil {
		if len(term.Etc()) > 0 {
			cerr = base.Print(t, term.Etc())
			if cerr != nil {
				return false, cerr
			}
		}
		return false, nil
	}
	return false, cerr
}

func (c *luaCmd) pushContext(r *rt.Runtime) {
	r.PushContext(rt.RuntimeContextDef{
		HardLimits: rt.RuntimeResources{
			Cpu:    c.cpuLimit,
			Memory: c.memLimit,
		},
		RequiredFlags:  c.complianceFlags,
		MessageHandler: debuglib.Traceback,
	})
}

type execFlags []string

func (e *execFlags) String() string {
	return strings.Join(*e, "; ")
}

func (e *execFlags) Set(value string) error {
	*e = append(*e, value)
	return nil
}
