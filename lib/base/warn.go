package base

import (
	"errors"
	"fmt"

	rt "github.com/arnodel/golua/runtime"
)

func warn(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := c.Etc()
	if len(args) == 0 {
		return nil, errors.New("bad argument #1 (value needed)")
	}
	msgs := make([]string, len(args))
	for i, v := range args {
		s, ok := v.ToString()
		if !ok {
			return nil, fmt.Errorf("bad argument #%d (string expected)", i+1)
		}
		msgs[i] = s
	}
	t.Warn(msgs...)
	return c.Next(), nil
}
