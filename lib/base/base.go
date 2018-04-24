package base

import (
	"errors"
	"fmt"

	rt "github.com/arnodel/golua/runtime"
)

func tostring(t *rt.Thread, args []rt.Value, next rt.Continuation) error {
	if len(args) == 0 {
		return errors.New("tostring needs 1 argument at least")
	}
	v := args[0]
	err, ok := rt.Metacall(t, v, "__tostring", []rt.Value{v}, next)
	if ok {
		return err
	}
	s, ok := rt.ToString(v)
	if !ok {
		s = rt.String(fmt.Sprintf("%s: <addr>", rt.Type(v)))
	}
	next.Push(s)
	return nil
}

func print(t *rt.Thread, args []rt.Value, next rt.Continuation) error {
	for i, v := range args {
		if i > 0 {
			fmt.Print("\t")
		}
		res := rt.NewTerminationWith(1, false)
		if err := tostring(t, []rt.Value{v}, res); err != nil {
			return err
		}
		fmt.Print(res.Get(0))
	}
	fmt.Print("\n")
	return nil
}

func Load(env *rt.Table) {
	env.Set(rt.String("print"), rt.GoFunction(print))
	env.Set(rt.String("tostring"), rt.GoFunction(tostring))
}
