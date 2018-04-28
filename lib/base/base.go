package base

import (
	"errors"
	"fmt"

	rt "github.com/arnodel/golua/runtime"
)

func Load(r *rt.Runtime) {
	env := r.GlobalEnv()
	// TODO: assert
	// TODO: collectgarbage (although what is there to do?
	// TODO: dofile
	rt.SetEnvFunc(env, "error", errorF)
	rt.SetEnv(env, "_G", env)
	rt.SetEnvFunc(env, "getmetatable", getmetatable)
	// TODO: ipairs
	rt.SetEnvFunc(env, "load", load)
	// TODO: load
	// TODO: loadfile
	// TODO: next
	// TODO: pairs
	rt.SetEnvFunc(env, "pcall", pcall)
	rt.SetEnvFunc(env, "print", print)
	rt.SetEnvFunc(env, "rawequal", rawequal)
	rt.SetEnvFunc(env, "rawget", rawget)
	rt.SetEnvFunc(env, "rawlen", rawlen)
	rt.SetEnvFunc(env, "rawset", rawset)
	// TODO: select
	rt.SetEnvFunc(env, "setmetatable", setmetatable)
	// TODO: tonumber
	rt.SetEnvFunc(env, "tostring", tostring)
	rt.SetEnvFunc(env, "type", typeString)
	rt.SetEnv(env, "_VERSION", rt.String("Golua 5.3"))
	// TODO: xpcall
}

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
			t.Stdout.Write([]byte{'\t'})
		}
		res := rt.NewTerminationWith(1, false)
		if err := tostring(t, []rt.Value{v}, res); err != nil {
			return err
		}
		t.Stdout.Write([]byte(res.Get(0).(rt.String)))
	}
	t.Stdout.Write([]byte{'\n'})
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
	// TODO: process level argument
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
	tbl, ok := args[0].(*rt.Table)
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

func getmetatable(t *rt.Thread, args []rt.Value, next rt.Continuation) error {
	if len(args) == 0 {
		return errors.New("getmetatable expects 1 argument")
	}
	next.Push(t.Metatable(args[0]))
	return nil
}

func setmetatable(t *rt.Thread, args []rt.Value, next rt.Continuation) error {
	if len(args) < 2 {
		return errors.New("setmetatable requires 2 arguments")
	}
	tbl, ok := args[0].(*rt.Table)
	if !ok {
		return errors.New("setmetatable: first argument must be a table")
	}
	if rt.RawGet(tbl.Metatable(), "__metatable") != nil {
		return errors.New("setmetatable: cannot set metatable")
	}
	if rt.IsNil(args[1]) {
		tbl.SetMetatable(nil)
	} else if meta, ok := args[1].(*rt.Table); ok {
		tbl.SetMetatable(meta)
	} else {
		return errors.New("setmetatable: second argument must be a table")
	}
	next.Push(args[0])
	return nil
}

func load(t *rt.Thread, args []rt.Value, next rt.Continuation) error {
	if len(args) == 0 {
		return errors.New("load requires 1 argument")
	}
	chunkArg := args[0]
	var chunk []byte
	chunkName := ""
	chunkMode := "bt"
	chunkEnv := t.GlobalEnv()
	switch x := chunkArg.(type) {
	case rt.String:
		chunk = []byte(x)
		chunkName = "chunk"
	default:
		return errors.New("load: chunk must be a string")
	}
	if len(args) >= 2 {
		name, ok := args[1].(rt.String)
		if !ok {
			return errors.New("load: chunkname must be as string")
		}
		chunkName = string(name)
	}
	if len(args) >= 3 {
		mode, ok := args[2].(rt.String)
		if !ok {
			return errors.New("load: mode must be a string")
		}
		chunkMode = string(mode)
	}
	if len(args) >= 4 {
		env, ok := args[3].(*rt.Table)
		if !ok {
			return errors.New("load: env must be a table")
		}
		chunkEnv = env
	}
	// TODO: use those
	_, _ = chunkName, chunkMode
	clos, err := rt.CompileLuaChunk(chunk, chunkEnv)
	if err != nil {
		return err
	}
	next.Push(clos)
	return nil
}
