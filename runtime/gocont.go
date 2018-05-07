package runtime

// GoCont implements Cont for functions written in Go.
type GoCont struct {
	f     func(*Thread, *GoCont) (Cont, *Error)
	next  Cont
	args  []Value
	etc   *[]Value
	nArgs int
}

func NewGoCont(f *GoFunction) *GoCont {
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
		args: args,
		etc:  etc,
	}
}

// Push implements Cont.Push.
func (c *GoCont) Push(v Value) {
	if c.next == nil {
		var ok bool
		c.next, ok = v.(Cont)
		if !ok {
			panic("First push must be a continuation")
		}
	} else if c.nArgs < len(c.args) {
		c.args[c.nArgs] = v
		c.nArgs++
	} else if c.etc != nil {
		*c.etc = append(*c.etc, v)
	}
}

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

func (c *GoCont) Next() Cont {
	return c.next
}

func (c *GoCont) NArgs() int {
	return c.nArgs
}

func (c *GoCont) Arg(n int) Value {
	return c.args[n]
}

func (c *GoCont) Args() []Value {
	return c.args[:c.nArgs]
}

func (c *GoCont) Etc() []Value {
	if c.etc == nil {
		return nil
	}
	return *c.etc
}

func (c *GoCont) Check1Arg() *Error {
	if c.nArgs == 0 {
		return NewErrorS("1 argument needed")
	}
	return nil
}

func (c *GoCont) CheckNArgs(n int) *Error {
	if c.nArgs < n {
		return NewErrorF("%d arguments needed", n)
	}
	return nil
}

func (c *GoCont) StringArg(n int) (String, *Error) {
	s, ok := c.Arg(n).(String)
	if !ok {
		return "", NewErrorF("#%d must be a string", n+1)
	}
	return s, nil
}

func (c *GoCont) CallableArg(n int) (Callable, *Error) {
	f, ok := c.Arg(n).(Callable)
	if !ok {
		return nil, NewErrorF("#%d must be a callable", n+1)
	}
	return f, nil
}

func (c *GoCont) ThreadArg(n int) (*Thread, *Error) {
	t, ok := c.Arg(n).(*Thread)
	if !ok {
		return nil, NewErrorF("#%d must be a callable", n+1)
	}
	return t, nil
}

func (c *GoCont) IntArg(n int) (Int, *Error) {
	i, tp := ToInt(c.Arg(n))
	if tp != IsInt {
		return 0, NewErrorF("#%d must be an integer", n+1)
	}
	return i, nil
}

func (c *GoCont) TableArg(n int) (*Table, *Error) {
	t, ok := c.Arg(n).(*Table)
	if !ok {
		return nil, NewErrorF("#%d must be a table", n+1)
	}
	return t, nil
}
