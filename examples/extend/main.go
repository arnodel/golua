package main

import (
	"os"

	"github.com/arnodel/golua/lib/base"
	rt "github.com/arnodel/golua/runtime"
)

// This is the Go function that we are going to call from Lua. Its inputs are:
//
// - t: the thread the function is running in.
//
// - c: the go continuation that represents the context the function is called
//      in.  It contains the arguments to the function and the next continuation
//      (the one which receives the values computed by this function).
//
// It returns the next continuation on success, else an error.
func addints(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var x, y int64

	// First check there are two arguments
	err := c.CheckNArgs(2)
	if err == nil {
		// Ok then try to convert the first argument to a lua integer.
		x, err = c.IntArg(0)
	}
	if err == nil {
		// Ok then try to convert the first argument to a lua integer.
		y, err = c.IntArg(1)
	}
	if err != nil {
		// Some error occurred, we return it in our context
		return nil, err
	}
	// Arguments parsed!  First get the next continuation.
	next := c.Next()

	// Then compute the result and push it to the continuation.
	t.Push1(next, rt.IntValue(x+y))

	// Finally return the next continuation.
	return next, nil

	// Note: the last 3 steps could have been written as:
	// return c.PushingNext(x + y), nil
}

func main() {
	// First we obtain a new Lua runtime which outputs to stdout
	r := rt.New(os.Stdout)

	// Load the basic library into the runtime (we need print)
	base.Load(r)

	// Then we add our addints function to the global environment of the
	// runtime.
	r.SetEnvGoFunc(r.GlobalEnv(), "addints", addints, 2, false)

	// This is the chunk we want to run.  It calls the addints function.
	source := []byte(`print("hello", addints(40, 2))`)

	// Compile the chunk.
	chunk, _ := r.CompileAndLoadLuaChunk("test", source, rt.TableValue(r.GlobalEnv()))

	// Run the chunk in the runtime's main thread.  It should output 42!
	_, _ = rt.Call1(r.MainThread(), rt.FunctionValue(chunk))
}
