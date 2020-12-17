package stringlib

import (
	"bytes"

	rt "github.com/arnodel/golua/runtime"
)

func dump(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	cl, err := c.ClosureArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	strip := false
	if c.NArgs() >= 2 {
		strip = rt.Truth(c.Arg(1))
	}
	// TODO: support strip
	_ = strip
	var w bytes.Buffer
	if err := rt.WriteConst(&w, rt.CodeValue(cl.Code.RefactorConsts())); err != nil {
		return nil, rt.NewError(rt.StringValue(err.Error())).AddContext(c)
	}
	return c.PushingNext(rt.StringValue(w.String())), nil
}
