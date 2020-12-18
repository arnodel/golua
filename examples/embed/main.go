package main

import (
	"fmt"
	"os"

	rt "github.com/arnodel/golua/runtime"
)

func main() {
	// First we obtain a new Lua runtime which outputs to stdout
	r := rt.New(os.Stdout)

	// This is the chunk we want to run.  It returns an adding function.
	source := []byte(`return function(x, y) return x + y end`)

	// Compile the chunk. Note that compiling doesn't require a runtime.
	chunk, _ := rt.CompileAndLoadLuaChunk("test", source, r.GlobalEnv())

	// Run the chunk in the runtime's main thread.  Its output is the Lua adding
	// function.
	add, _ := rt.Call1(r.MainThread(), rt.FunctionValue(chunk))

	// Now, run the Lua function in the main thread.
	sum, _ := rt.Call1(r.MainThread(), add, rt.IntValue(40), rt.IntValue(2))

	// --> 42
	fmt.Println(sum)
}
