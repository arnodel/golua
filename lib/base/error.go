package base

import (
	"fmt"

	rt "github.com/arnodel/golua/runtime"
)

func errorF(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if c.NArgs() == 0 {
		return nil, rt.NewError(rt.NilValue)
	}
	next := c.Next()
	if c.NArgs() >= 2 {
		level, err := c.IntArg(1)
		if err != nil {
			return nil, err
		}
		if level < 1 {
			next = nil
		}
		for level > 1 && next != nil {
			next = next.Next()
			level--
		}
	}
	msg := c.Arg(0)
	if next != nil {
		s, ok := msg.TryString()
		if ok {
			info := next.DebugInfo()
			if info != nil {
				msg = rt.StringValue(fmt.Sprintf("%s:%d: %s", info.Source, info.CurrentLine, s))
			}
		}
	}
	return nil, rt.NewError(msg)
}
