// Package contlib is the Lua library "cont", which gives Lua programs
// first-class continuations.  It is not part of Lua: golua's runtime is
// built on continuations rather than on a stack, and this library is how a
// Lua program gets hold of one.
//
// The library has one function:
//
//	cont.callcc(f)
//
// calls f(k) and returns what f returns.  k is a function that stands for
// the rest of the program from the call of callcc: calling k(...) makes that
// call of callcc return the arguments of k.  k can be called from inside f,
// which leaves f, or after callcc has returned, which makes it return again,
// and any number of times.
//
// A local variable that no function captures and assigns is part of what a
// continuation holds: it has the value it had at the call of callcc each
// time k is called.  Tables, and variables that a function captures and
// assigns, are shared with the rest of the program and keep their changes.
//
// A continuation holds the Lua functions that are running, not the Go
// functions that have called them: it cannot be called once the pcall,
// metamethod call or coroutine it was captured in has returned, nor from
// inside another one, nor from another coroutine.  Each is an error.
//
// To-be-closed variables and continuations do not work together yet:
// calling a continuation does not close the variables of the blocks it
// leaves at that point.
package contlib

import (
	"github.com/arnodel/golua/lib/packagelib"
	rt "github.com/arnodel/golua/runtime"
)

// LibLoader allows loading the cont library.  Its OnRequire method gives a
// loader that makes it available to require("cont") instead, which is how
// lib.LoadAll provides it.
var LibLoader = packagelib.Loader{
	Load: load,
	Name: "cont",
}

func load(r *rt.Runtime) (rt.Value, func()) {
	pkg := rt.NewTable()
	callcc := rt.NewCallCC()
	callcc.SolemnlyDeclareCompliance(rt.ComplyCpuSafe | rt.ComplyMemSafe | rt.ComplyTimeSafe | rt.ComplyIoSafe)
	r.SetEnv(pkg, "callcc", rt.FunctionValue(callcc))
	return rt.TableValue(pkg), nil
}
