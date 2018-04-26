package base

import (
	"errors"
	"fmt"

	rt "github.com/arnodel/golua/runtime"
)

// func assert(t *rt.Thread, args []rt.Value, next rt.Continuation) error {
// 	if len(args) == 0 {
// 		return errors.New("assert needs at least one argument")
// 	}
// 	v := args[0]
// 	if len(args) >= 2 {

// 	}
// }

func tostring(t *rt.Thread, args []rt.Value, next rt.Continuation) error {
	if len(args) == 0 {
		return errors.New("tostring needs 1 argument at least")
	}
	v := args[0]
	err, ok := rt.Metacall(t, v, "__tostring", []rt.Value{v}, next)
	if ok {
		return err
	}
	s, ok := rt.AsString(v)
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

func typeString(t *rt.Thread, args []rt.Value, next rt.Continuation) error {
	if len(args) == 0 {
		return errors.New("type needs 1 argument")
	}
	next.Push(rt.Type(args[0]))
	return nil
}

func errorF(t *rt.Thread, args []rt.Value, next rt.Continuation) error {
	if len(args) == 0 {
		return errors.New("error needs 1 argument")
	}
	return rt.ErrorFromValue(args[0])
}

func pcall(t *rt.Thread, args []rt.Value, next rt.Continuation) error {
	if len(args) == 0 {
		return errors.New("pcall needs 1 argument")
	}
	res := rt.NewTerminationWith(0, true)
	if err := rt.Call(t, args[0], args[1:], res); err != nil {
		next.Push(rt.Bool(false))
		next.Push(rt.ValueFromError(err))
	} else {
		next.Push(rt.Bool(true))
		rt.Push(next, res.Etc()...)
	}
	return nil
}

func rawequal(t *rt.Thread, args []rt.Value, next rt.Continuation) error {
	if len(args) < 2 {
		return errors.New("rawequal requires 2 arguments")
	}
	res, _ := rt.RawEqual(args[0], args[1])
	next.Push(rt.Bool(res))
	return nil
}

func rawget(t *rt.Thread, args []rt.Value, next rt.Continuation) error {
	if len(args) < 2 {
		return errors.New("rawget requires 2 arguments")
	}
	tbl, ok := args[0].(*rt.Table)
	if !ok {
		return errors.New("rawget: first argument must be a table")
	}
	next.Push(rt.RawGet(tbl, args[1]))
	return nil
}

func rawset(t *rt.Thread, args []rt.Value, next rt.Continuation) error {
	if len(args) < 3 {
		return errors.New("rawset requires 3 arguments")
	}
	tlb, ok := args[0].(*rt.Table)
	if !ok {
		return errors.New("rawset: first argument must be a table")
	}
	if rt.IsNil(args[1]) {
		return errors.New("rawset: second argument must not be nil")
	}
	tbl.Set(args[1], args[2])
	next.Push(args[0])
	return nil
}

func rawlen(t *rt.Thread, args []rt.Value, next rt.Continuation) error {
	if len(args) == 0 {
		return errors.New("rawlen needs 1 argument")
	}
	switch x := args[0].(type) {
	case rt.String:
		next.Push(rt.Int(len(x)))
		return nil
	case *rt.Table:
		next.Push(x.Len())
		return nil
	}
	return errors.New("rawlen requires a string or table")
}

func Load(env *rt.Table) {
	rt.SetEnvFunc(env, "print", print)
	rt.SetEnvFunc(env, "tostring", tostring)
	rt.SetEnvFunc(env, "type", typeString)
	rt.SetEnvFunc(env, "pcall", pcall)
	rt.SetEnvFunc(env, "error", errorF)
	rt.SetEnvFunc(env, "rawequal", rawequal)
	rt.SetEnvFunc(env, "rawget", rawget)
	rt.SetEnvFunc(env, "rawset", rawset)
	rt.SetEnvFunc(env, "rawlen", rawlen)
	env.Set("_G", env)
}
