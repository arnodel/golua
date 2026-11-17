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
// functions that have called them, such as pcall, a metamethod call, or
// table.sort calling its comparison function.  Calling a continuation from
// under one of these leaves it: the Go function returns to no one, and the
// to-be-closed variables of the Lua functions it had called are closed.  So
// a continuation is a way out of a pcall.  It is not a way back in: it is an
// error to call a continuation once the pcall, metamethod call or coroutine
// it was captured in has returned, and to call it from another coroutine.
// Nor is it a way out of a context (runtime.callcontext), which ends with an
// error instead.
//
// Calling a continuation leaves blocks, as break and return do, and their
// to-be-closed variables are closed, the innermost first.  The exception is
// a block in which a continuation was captured: it can be entered again, by
// calling that continuation, so its variable stays pending, as that of a
// coroutine which has yielded does, and is closed when the block ends.  A
// block that is entered several times ends several times, and its variable
// is closed once.
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

	rt.SolemnlyDeclareCompliance(
		rt.ComplyCpuSafe|rt.ComplyMemSafe|rt.ComplyTimeSafe|rt.ComplyIoSafe,

		r.SetEnvGoFunc(pkg, "callcc", callcc, 1, false),
	)

	return rt.TableValue(pkg), nil
}

// callcc(f) calls f(k), where k is the continuation of the call of callcc.
func callcc(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	k, next := t.CaptureCont(c.Next())

	// f returns where callcc would: this is a tail call.
	fc, err := rt.Continue(t, c.Arg(0), next)
	if err != nil {
		return nil, err
	}
	fc.Push(t.Runtime, k)
	return fc, nil
}
