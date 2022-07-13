package runtime

import (
	"io"
)

// DoInContext creates a *Runtime r with the constraints from the
// RuntimeContextDef def, calls f(r) and returns two things:  A RuntimeContext
// instance ctx whose status tells us the outcome (done, killed or error) and in
// case of error, a non-nil error value.
func DoInContext(f func(r *Runtime) error, def RuntimeContextDef, stdout io.Writer, opts ...RuntimeOption) (ctx RuntimeContext, err error) {
	r := New(stdout, opts...)
	defer r.Close(nil)
	ctx, err = r.MainThread().CallContext(def, func() error {
		return f(r)
	})
	return
}

// RunChunk1 runs source code in a runtime constrained by def.  It returns the
// value returned by the chunk and possibly an error if execution failed
// (including when the runtime was killed).
func RunChunk1(source []byte, def RuntimeContextDef, stdout io.Writer, opts ...RuntimeOption) (v Value, err error) {
	_, err = DoInContext(func(r *Runtime) error {
		env := TableValue(r.GlobalEnv())
		chunk, err := r.CompileAndLoadLuaChunk("code", source, env)
		if err != nil {
			return err
		}
		v, err = Call1(r.MainThread(), FunctionValue(chunk))
		return err
	}, def, stdout, opts...)
	return v, err
}
