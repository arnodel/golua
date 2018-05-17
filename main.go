package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/arnodel/golua/lib/base"
	"github.com/arnodel/golua/lib/coroutine"
	"github.com/arnodel/golua/lib/packagelib"
	"github.com/arnodel/golua/runtime"
)

func main() {
	flag.Parse()
	var chunkName string
	var chunk []byte
	var err error
	switch flag.NArg() {
	case 0:
		chunkName = "<stdin>"
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
	r := runtime.New(os.Stdout)
	base.Load(r)
	coroutine.Load(r)
	packagelib.Load(r)
	t := r.MainThread()
	clos, err := runtime.CompileLuaChunk(chunkName, chunk, r.GlobalEnv())
	if err != nil {
		fatal("Error parsing %s: %s", chunkName, err)
	}
	cerr := runtime.Call(t, clos, nil, runtime.NewTerminationWith(0, false))
	if cerr != nil {
		fatal("Error running %s: %s", chunkName, cerr)
	}
}

func fatal(tpl string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, tpl, args...)
	os.Exit(1)
}
