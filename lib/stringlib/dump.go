package stringlib

import (
	"bytes"

	rt "github.com/arnodel/golua/runtime"
)

func dump(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	cl, err := c.ClosureArg(0)
	if err != nil {
		return nil, err
	}
	strip := false
	if c.NArgs() >= 2 {
		strip = rt.Truth(c.Arg(1))
	}
	// TODO: support strip
	_ = strip
	var w bytes.Buffer
	code := t.RefactorCodeConsts(cl.Code)
	used, mErr := rt.MarshalConst(&w, rt.CodeValue(code), t.LinearUnused(10))
	// This will cause a panic if MarshalConst was interupted, so no need to
	// worry about the rest of this codepath in this case.
	t.LinearRequire(10, used)
	if err != nil {
		return nil, mErr
	}
	return c.PushingNext1(t.Runtime, rt.StringValue(w.String())), nil
}
