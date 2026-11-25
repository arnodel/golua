package runtime

import (
	"errors"
	"fmt"
	"unsafe"
)

// GoCont implements Cont for functions written in Go.
type GoCont struct {
	*GoFunction
	next  Cont
	args  []Value
	etc   []Value // Only for functions with hasEtc; from argsPool, released with c
	nArgs int
}

var _ Cont = (*GoCont)(nil)

// NewGoCont returns a new pointer to GoCont for the given GoFunction and Cont.
func NewGoCont(t *Thread, f *GoFunction, next Cont) *GoCont {
	var args, etc []Value
	if f.nArgs > 0 {
		t.RequireArrSize(unsafe.Sizeof(Value{}), f.nArgs)
		args = t.argsPool.get(f.nArgs)
	}
	if f.hasEtc {
		etc = t.argsPool.get(0)
	}
	t.RequireSize(unsafe.Sizeof(GoCont{}))
	cont := t.goContPool.get()
	*cont = GoCont{
		GoFunction: f,
		args:       args,
		etc:        etc,
		next:       next,
	}
	return cont
}

// clone returns a new GoCont for the same call as c, with the arguments that
// have been pushed into c, and returning to next.
func (c *GoCont) clone(t *Thread, next Cont) *GoCont {
	d := NewGoCont(t, c.GoFunction, next)
	d.PushEtc(t.Runtime, c.args[:c.nArgs])
	d.PushEtc(t.Runtime, c.etc)
	return d
}

// thawNext replaces a frozen next continuation with a copy that can run (see
// frozenCont), before the function runs: it pushes its results into the next
// continuation.  This is the test, which is inlined where a Go function is
// entered; replaceFrozenNext does the rest.
func (c *GoCont) thawNext(t *Thread) {
	if fc, ok := c.next.(*frozenCont); ok {
		c.replaceFrozenNext(t, fc)
	}
}

// replaceFrozenNext makes the next continuation of c, which is the frozen
// frame fc, a copy of it that can run.  A function that makes no use of its
// next continuation keeps the frozen frame: the copy would be made for
// nothing.
//
// It must not be inlined into thawNext, which would then be too big to be
// inlined itself.
//
//go:noinline
func (c *GoCont) replaceFrozenNext(t *Thread, fc *frozenCont) {
	if !c.ignoresNext {
		c.next = fc.thaw(t)
	}
}

// Push implements Cont.Push.
func (c *GoCont) Push(r *Runtime, v Value) {
	if c.nArgs < len(c.args) {
		c.args[c.nArgs] = v
		c.nArgs++
	} else if c.hasEtc {
		r.RequireSize(unsafe.Sizeof(Value{}))
		c.etc = append(c.etc, v)
	}
}

// PushingNext is convenient when implementing go functions.  It pushes the
// given values to c.Next() and returns it.
func (c *GoCont) PushingNext(r *Runtime, vals ...Value) Cont {
	next := c.Next()
	next.PushEtc(r, vals)
	return next
}

// PushingNext1 is convenient when implementing go functions.  It pushes the
// given value to c.Next() and returns it.
func (c *GoCont) PushingNext1(r *Runtime, val Value) Cont {
	next := c.Next()
	next.Push(r, val)
	return next
}

// PushEtc pushes a slice of values to the continutation.
func (c *GoCont) PushEtc(r *Runtime, etc []Value) {
	if c.nArgs < len(c.args) {
		for i, v := range etc {
			c.args[c.nArgs] = v
			c.nArgs++
			if c.nArgs == len(c.args) {
				etc = etc[i+1:]
				goto FillEtc
			}
		}
		return
	}
FillEtc:
	if !c.hasEtc {
		return
	}
	r.RequireArrSize(unsafe.Sizeof(Value{}), len(etc))
	c.etc = append(c.etc, etc...)
}

// RunInThread implements Cont.RunInThread.
func (c *GoCont) RunInThread(t *Thread) (next Cont, err error) {
	if err := t.CheckRequiredFlags(c.safetyFlags); err != nil {
		return nil, err
	}
	t.RequireCPU(1)
	c.thawNext(t) // On a line that was empty: the profile counts lines
	t.goFunctionCallDepth++
	defer func() { t.goFunctionCallDepth-- }()

	if t.goFunctionCallDepth > maxGoFunctionCallDepth {
		return nil, errors.New("stack overflow")
	}
	next, err = c.f(t, c)
	if err == errLightYield {
		// The thread is suspended in the function, which has not returned:
		// it is run again when the thread is resumed.
		return nil, err
	}
	if t.DebugHookFlags&HookFlagReturn != 0 && (err != nil || next == c.next) {
		// A function that hands over to another continuation has not
		// returned: that is for what it handed over to (a protected call).
		_ = t.triggerReturn(t, c)
	}

	if err != nil {
		// If there is an error, c is still potentially needed for error
		// handling, so do not return it to the pool.  It will get GCed when no
		// longer referenced, so it's OK.
		return
	}
	if c.args != nil {
		t.ReleaseArrSize(unsafe.Sizeof(Value{}), c.nArgs)
		t.argsPool.release(c.args)
	}
	if c.hasEtc {
		t.ReleaseArrSize(unsafe.Sizeof(Value{}), len(c.etc))
		t.argsPool.release(c.etc)
	}
	t.ReleaseSize(unsafe.Sizeof(GoCont{}))
	t.goContPool.release(c)
	return
}

// Next returns the next continuation.
func (c *GoCont) Next() Cont {
	return c.next
}

// Parent returns the continuation's parent.
func (c *GoCont) Parent() Cont {
	return c.next
}

// DebugInfo returns c's debug info.
func (c *GoCont) DebugInfo() *DebugInfo {
	name := c.name
	if name == "" {
		name = "<go function>"
	}
	return &DebugInfo{
		Source:      "[Go]",
		CurrentLine: 0,
		Name:        name,
	}
}

// NArgs returns the number of args pushed to the continuation.
func (c *GoCont) NArgs() int {
	return c.nArgs
}

// Arg returns the n-th arg of the continuation.  It doesn't do any range check!
func (c *GoCont) Arg(n int) Value {
	return c.args[n]
}

// Args returns the slice of args pushed to the continuation.
func (c *GoCont) Args() []Value {
	return c.args[:c.nArgs]
}

// Etc returns the etc args pushed to the continuation, if the function takes
// them.  The slice is reused once the function returns, so it must not be
// kept beyond that.
func (c *GoCont) Etc() []Value {
	return c.etc
}

// Check1Arg returns a non-nil error if the continuation doesn't have at least
// one arg.
func (c *GoCont) Check1Arg() error {
	if c.nArgs == 0 {
		return errors.New("bad argument #1 (value needed)")
	}
	return nil
}

// CheckNArgs returns a non-nil error if the continuation doesn't have at least
// n args.
func (c *GoCont) CheckNArgs(n int) error {
	if c.nArgs < n {
		return fmt.Errorf("%d arguments needed", n)
	}
	return nil
}

// StringArg returns the n-th argument as a string if possible, otherwise a
// non-nil error.  A number is converted to a string, as described in §3.4.3
// of the Lua manual.  No range check!
func (c *GoCont) StringArg(n int) (string, error) {
	s, ok := c.Arg(n).ToString()
	if !ok {
		return "", fmt.Errorf("#%d must be a string", n+1)
	}
	return s, nil
}

// BoolArg returns the n-th argument as a string if possible, otherwise a
// non-nil error.  No range check!
func (c *GoCont) BoolArg(n int) (bool, error) {
	arg := c.Arg(n)
	if arg.IsNil() {
		return false, nil
	}
	b, ok := arg.TryBool()
	if !ok {
		return false, fmt.Errorf("#%d must be a boolean", n+1)
	}
	return b, nil
}

// CallableArg returns the n-th argument as a callable if possible, otherwise a
// non-nil error.  No range check!
func (c *GoCont) CallableArg(n int) (Callable, error) {
	f, ok := c.Arg(n).TryCallable()
	if !ok {
		return nil, fmt.Errorf("#%d must be a callable", n+1)
	}
	return f, nil
}

// ClosureArg returns the n-th argument as a closure if possible, otherwise a
// non-nil error.  No range check!
func (c *GoCont) ClosureArg(n int) (*Closure, error) {
	f, ok := c.Arg(n).TryClosure()
	if !ok {
		return nil, fmt.Errorf("#%d must be a lua function", n+1)
	}
	return f, nil
}

// ThreadArg returns the n-th argument as a thread if possible, otherwise a
// non-nil error.  No range check!
func (c *GoCont) ThreadArg(n int) (*Thread, error) {
	t, ok := c.Arg(n).TryThread()
	if !ok {
		return nil, fmt.Errorf("#%d must be a thread", n+1)
	}
	return t, nil
}

// IntArg returns the n-th argument as an Int if possible, otherwise a
// non-nil error.  No range check!
func (c *GoCont) IntArg(n int) (int64, error) {
	i, ok := ToInt(c.Arg(n))
	if !ok {
		return 0, fmt.Errorf("#%d must be an integer", n+1)
	}
	return i, nil
}

// FloatArg returns the n-th argument as a Float if possible, otherwise a
// non-nil error.  No range check!
func (c *GoCont) FloatArg(n int) (float64, error) {
	x, ok := ToFloat(c.Arg(n))
	if !ok {
		return 0, fmt.Errorf("#%d must be a number", n+1)
	}
	return x, nil
}

// TableArg returns the n-th argument as a table if possible, otherwise a
// non-nil error.  No range check!
func (c *GoCont) TableArg(n int) (*Table, error) {
	t, ok := c.Arg(n).TryTable()
	if !ok {
		return nil, fmt.Errorf("#%d must be a table", n+1)
	}
	return t, nil
}

// UserDataArg returns the n-th argument as a UserData if possible, otherwise a
// non-nil error.  No range check!
func (c *GoCont) UserDataArg(n int) (*UserData, error) {
	t, ok := c.Arg(n).TryUserData()
	if !ok {
		return nil, fmt.Errorf("#%d must be userdata", n+1)
	}
	return t, nil
}
