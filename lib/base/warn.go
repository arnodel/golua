package base

import rt "github.com/arnodel/golua/runtime"

func warn(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	args := c.Etc()
	if len(args) == 0 {
		return nil, rt.NewErrorS("bad argument #1 (value needed)")
	}
	msgs := make([]string, len(args))
	for i, v := range args {
		s, ok := v.ToString()
		if !ok {
			return nil, rt.NewErrorF("bad argument #%d (string expected)", i+1)
		}
		msgs[i] = s
	}
	t.Warn(msgs...)
	return c.Next(), nil
}
