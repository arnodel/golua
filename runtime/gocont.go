package runtime

// GoCont implements Cont for functions written in Go.
type GoCont struct {
	f     func(*Thread, *GoCont) (Cont, *Error)
	name  string
	next  Cont
	args  []Value
	etc   *[]Value
	nArgs int
}

// NewGoCont returns a new pointer to GoCont for the given GoFunction and Cont.
func NewGoCont(f *GoFunction, next Cont) *GoCont {
	var args []Value
	var etc *[]Value
	if f.nArgs > 0 {
		args = make([]Value, f.nArgs)
	}
	if f.hasEtc {
		etc = new([]Value)
	}
	return &GoCont{
		f:    f.f,
		name: f.name,
		args: args,
		etc:  etc,
		next: next,
	}
}

// Push implements Cont.Push.
func (c *GoCont) Push(v Value) {
	if c.nArgs < len(c.args) {
		c.args[c.nArgs] = v
		c.nArgs++
	} else if c.etc != nil {
		*c.etc = append(*c.etc, v)
	}
}

// PushingNext is convenient when implementing go functions.  It pushes the
// given values to c.Next() and returns it.
func (c *GoCont) PushingNext(vals ...Value) Cont {
	next := c.Next()
	next.PushEtc(vals)
	return next
}

// PushEtc pushes a slice of values to the continutation. TODO: find why this is
// not used.
func (c *GoCont) PushEtc(etc []Value) {
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
	if c.etc == nil {
		return
	}
	*c.etc = append(*c.etc, etc...)
}

// RunInThread implements Cont.RunInThread
func (c *GoCont) RunInThread(t *Thread) (Cont, *Error) {
	return c.f(t, c)
}

// Next implements Cont.Next.
func (c *GoCont) Next() Cont {
	return c.next
}

// DebugInfo implements Cont.DebugInfo.
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

// Etc returns the etc args pushed to the continuation they exist.
func (c *GoCont) Etc() []Value {
	if c.etc == nil {
		return nil
	}
	return *c.etc
}

// Check1Arg returns a non-nil *Error if the continuation doesn't have at least
// one arg.
func (c *GoCont) Check1Arg() *Error {
	if c.nArgs == 0 {
		return NewErrorS("bad argument #1 (value needed)")
	}
	return nil
}

// CheckNArgs returns a non-nil *Error if the continuation doesn't have at least
// n args.
func (c *GoCont) CheckNArgs(n int) *Error {
	if c.nArgs < n {
		return NewErrorF("%d arguments needed", n)
	}
	return nil
}

// StringArg returns the n-th argument as a string if possible, otherwise a
// non-nil *Error.  No range check!
func (c *GoCont) StringArg(n int) (string, *Error) {
	s, ok := c.Arg(n).TryString()
	if !ok {
		return "", NewErrorF("#%d must be a string", n+1)
	}
	return s, nil
}

// CallableArg returns the n-th argument as a callable if possible, otherwise a
// non-nil *Error.  No range check!
func (c *GoCont) CallableArg(n int) (Callable, *Error) {
	f, ok := c.Arg(n).TryCallable()
	if !ok {
		return nil, NewErrorF("#%d must be a callable", n+1)
	}
	return f, nil
}

// ClosureArg returns the n-th argument as a closure if possible, otherwise a
// non-nil *Error.  No range check!
func (c *GoCont) ClosureArg(n int) (*Closure, *Error) {
	f, ok := c.Arg(n).TryClosure()
	if !ok {
		return nil, NewErrorF("#%d must be a lua function", n+1)
	}
	return f, nil
}

// ThreadArg returns the n-th argument as a thread if possible, otherwise a
// non-nil *Error.  No range check!
func (c *GoCont) ThreadArg(n int) (*Thread, *Error) {
	t, ok := c.Arg(n).TryThread()
	if !ok {
		return nil, NewErrorF("#%d must be a thread", n+1)
	}
	return t, nil
}

// IntArg returns the n-th argument as an Int if possible, otherwise a
// non-nil *Error.  No range check!
func (c *GoCont) IntArg(n int) (int64, *Error) {
	i, ok := ToInt(c.Arg(n))
	if !ok {
		return 0, NewErrorF("#%d must be an integer", n+1)
	}
	return i, nil
}

// FloatArg returns the n-th argument as a Float if possible, otherwise a
// non-nil *Error.  No range check!
func (c *GoCont) FloatArg(n int) (float64, *Error) {
	x, ok := ToFloat(c.Arg(n))
	if !ok {
		return 0, NewErrorF("#%d must be a number", n+1)
	}
	return x, nil
}

// TableArg returns the n-th argument as a table if possible, otherwise a
// non-nil *Error.  No range check!
func (c *GoCont) TableArg(n int) (*Table, *Error) {
	t, ok := c.Arg(n).TryTable()
	if !ok {
		return nil, NewErrorF("#%d must be a table", n+1)
	}
	return t, nil
}

// UserDataArg returns the n-th argument as a UserData if possible, otherwise a
// non-nil *Error.  No range check!
func (c *GoCont) UserDataArg(n int) (*UserData, *Error) {
	t, ok := c.Arg(n).TryUserData()
	if !ok {
		return nil, NewErrorF("#%d must be userdata", n+1)
	}
	return t, nil
}
