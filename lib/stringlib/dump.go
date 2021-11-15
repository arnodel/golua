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
	code := t.RefactorCodeConsts(cl.Code)
	if used, err := rt.MarshalConst(&w, rt.CodeValue(code), t.UnusedMem()); err != nil {
		return nil, rt.NewError(rt.StringValue(err.Error())).AddContext(c)
	} else {
		// This will cause a panic if MarshalConst was interupted, so no need to
		// worry about the rest of this codepath in this case.
		t.RequireMem(used)
	}
	return c.PushingNext1(t.Runtime, rt.StringValue(w.String())), nil
}
