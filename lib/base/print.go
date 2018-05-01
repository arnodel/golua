package base

import rt "github.com/arnodel/golua/runtime"

func print(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	for i, v := range c.Etc() {
		if i > 0 {
			t.Stdout.Write([]byte{'\t'})
		}
		s, err := toString(t, v)
		if err != nil {
			return nil, err.AddContext(c)
		}
		t.Stdout.Write([]byte(s))
	}
	t.Stdout.Write([]byte{'\n'})
	return c.Next(), nil
}
