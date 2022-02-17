package base

import (
	"errors"

	rt "github.com/arnodel/golua/runtime"
)

func print(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	err := Print(t, c.Etc())
	if err != nil {
		return nil, err
	}
	return c.Next(), nil
}

func Print(t *rt.Thread, args []rt.Value) error {
	tostring := t.GlobalEnv().Get(rt.StringValue("tostring"))
	for i, v := range args {
		if i > 0 {
			t.Stdout.Write([]byte{'\t'})
		}
		vs, err := rt.Call1(t, tostring, v)
		if err != nil {
			return err
		}
		if s, ok := vs.TryString(); ok {
			t.Stdout.Write([]byte(s))
		} else {
			return errors.New("tostring must return a string")
		}
	}
	t.Stdout.Write([]byte{'\n'})
	return nil
}
