package base

import rt "github.com/arnodel/golua/runtime"

func print(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	err := Print(t, c.Etc())
	if err != nil {
		return nil, err.AddContext(c)
	}
	return c.Next(), nil
}

func Print(t *rt.Thread, args []rt.Value) *rt.Error {
	tostring := t.GlobalEnv().Get(rt.String("tostring"))
	for i, v := range args {
		if i > 0 {
			t.Stdout.Write([]byte{'\t'})
		}
		vs, err := rt.Call1(t, tostring, v)
		if err != nil {
			return err
		}
		if s, ok := vs.(rt.String); ok {
			t.Stdout.Write([]byte(s))
		} else {
			return rt.NewErrorS("tostring must return a string")
		}
	}
	t.Stdout.Write([]byte{'\n'})
	return nil
}
