package base

import rt "github.com/arnodel/golua/runtime"

func print(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	for i, v := range c.Etc() {
		if i > 0 {
			t.Stdout.Write([]byte{'\t'})
		}
		res := rt.NewTerminationWith(1, false)
		if _, err := tostring(t, []rt.Value{v}, res); err != nil {
			return c, err
		}
		t.Stdout.Write([]byte(res.Get(0).(rt.String)))
	}
	t.Stdout.Write([]byte{'\n'})
	return c.Next(), nil
}
