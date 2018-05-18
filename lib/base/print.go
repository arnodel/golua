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
	for i, v := range args {
		if i > 0 {
			t.Stdout.Write([]byte{'\t'})
		}
		s, err := toString(t, v)
		if err != nil {
			return err
		}
		t.Stdout.Write([]byte(s))
	}
	t.Stdout.Write([]byte{'\n'})
	return nil
}
