package base

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	rt "github.com/arnodel/golua/runtime"
)

func Load(r *rt.Runtime) {
	env := r.GlobalEnv()
	// TODO: assert
	// TODO: collectgarbage (although what is there to do?
	rt.SetEnvFunc(env, "dofile", dofile)
	rt.SetEnvFunc(env, "error", errorF)
	rt.SetEnv(env, "_G", env)
	rt.SetEnvFunc(env, "getmetatable", getmetatable)
	// TODO: ipairs
	rt.SetEnvFunc(env, "load", load)
	rt.SetEnvFunc(env, "loadfile", loadfile)
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
	rt.SetEnvFunc(env, "tonumber", tonumber)
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

func tostring(t *rt.Thread, args []rt.Value, next rt.Cont) (rt.Cont, error) {
	if len(args) == 0 {
		return nil, errors.New("tostring needs 1 argument at least")
	}
	v := args[0]
	err, ok := rt.Metacall(t, v, "__tostring", []rt.Value{v}, next)
	if ok {
		return nil, err
	}
	s, ok := rt.AsString(v)
	if !ok {
		s = rt.String(fmt.Sprintf("%s: <addr>", rt.Type(v)))
	}
	next.Push(s)
	return next, nil
}

func print(t *rt.Thread, args []rt.Value, next rt.Cont) (rt.Cont, error) {
	for i, v := range args {
		if i > 0 {
			t.Stdout.Write([]byte{'\t'})
		}
		res := rt.NewTerminationWith(1, false)
		if _, err := tostring(t, []rt.Value{v}, res); err != nil {
			return nil, err
		}
		t.Stdout.Write([]byte(res.Get(0).(rt.String)))
	}
	t.Stdout.Write([]byte{'\n'})
	return next, nil
}

func typeString(t *rt.Thread, args []rt.Value, next rt.Cont) (rt.Cont, error) {
	if len(args) == 0 {
		return nil, errors.New("type needs 1 argument")
	}
	next.Push(rt.Type(args[0]))
	return next, nil
}

func errorF(t *rt.Thread, args []rt.Value, next rt.Cont) (rt.Cont, error) {
	// TODO: process level argument
	if len(args) == 0 {
		return nil, errors.New("error needs 1 argument")
	}
	return nil, rt.ErrorFromValue(args[0])
}

func pcall(t *rt.Thread, args []rt.Value, next rt.Cont) (rt.Cont, error) {
	if len(args) == 0 {
		return nil, errors.New("pcall needs 1 argument")
	}
	res := rt.NewTerminationWith(0, true)
	if err := rt.Call(t, args[0], args[1:], res); err != nil {
		next.Push(rt.Bool(false))
		next.Push(rt.ValueFromError(err))
	} else {
		next.Push(rt.Bool(true))
		rt.Push(next, res.Etc()...)
	}
	return next, nil
}

func rawequal(t *rt.Thread, args []rt.Value, next rt.Cont) (rt.Cont, error) {
	if len(args) < 2 {
		return nil, errors.New("rawequal requires 2 arguments")
	}
	res, _ := rt.RawEqual(args[0], args[1])
	next.Push(rt.Bool(res))
	return next, nil
}

func rawget(t *rt.Thread, args []rt.Value, next rt.Cont) (rt.Cont, error) {
	if len(args) < 2 {
		return nil, errors.New("rawget requires 2 arguments")
	}
	tbl, ok := args[0].(*rt.Table)
	if !ok {
		return nil, errors.New("rawget: first argument must be a table")
	}
	next.Push(rt.RawGet(tbl, args[1]))
	return next, nil
}

func rawset(t *rt.Thread, args []rt.Value, next rt.Cont) (rt.Cont, error) {
	if len(args) < 3 {
		return nil, errors.New("rawset requires 3 arguments")
	}
	tbl, ok := args[0].(*rt.Table)
	if !ok {
		return nil, errors.New("rawset: first argument must be a table")
	}
	if rt.IsNil(args[1]) {
		return nil, errors.New("rawset: second argument must not be nil")
	}
	tbl.Set(args[1], args[2])
	next.Push(args[0])
	return next, nil
}

func rawlen(t *rt.Thread, args []rt.Value, next rt.Cont) (rt.Cont, error) {
	if len(args) == 0 {
		return nil, errors.New("rawlen needs 1 argument")
	}
	switch x := args[0].(type) {
	case rt.String:
		next.Push(rt.Int(len(x)))
		return next, nil
	case *rt.Table:
		next.Push(x.Len())
		return next, nil
	}
	return nil, errors.New("rawlen requires a string or table")
}

func getmetatable(t *rt.Thread, args []rt.Value, next rt.Cont) (rt.Cont, error) {
	if len(args) == 0 {
		return nil, errors.New("getmetatable expects 1 argument")
	}
	next.Push(t.Metatable(args[0]))
	return next, nil
}

func setmetatable(t *rt.Thread, args []rt.Value, next rt.Cont) (rt.Cont, error) {
	if len(args) < 2 {
		return nil, errors.New("setmetatable requires 2 arguments")
	}
	tbl, ok := args[0].(*rt.Table)
	if !ok {
		return nil, errors.New("setmetatable: first argument must be a table")
	}
	if rt.RawGet(tbl.Metatable(), "__metatable") != nil {
		return nil, errors.New("setmetatable: cannot set metatable")
	}
	if rt.IsNil(args[1]) {
		tbl.SetMetatable(nil)
	} else if meta, ok := args[1].(*rt.Table); ok {
		tbl.SetMetatable(meta)
	} else {
		return nil, errors.New("setmetatable: second argument must be a table")
	}
	next.Push(args[0])
	return next, nil
}

func load(t *rt.Thread, args []rt.Value, next rt.Cont) (rt.Cont, error) {
	if len(args) == 0 {
		return nil, errors.New("load requires 1 argument")
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
		return nil, errors.New("load: chunk must be a string")
	}
	if len(args) >= 2 {
		name, ok := args[1].(rt.String)
		if !ok {
			return nil, errors.New("load: chunkname must be as string")
		}
		chunkName = string(name)
	}
	if len(args) >= 3 {
		mode, ok := args[2].(rt.String)
		if !ok {
			return nil, errors.New("load: mode must be a string")
		}
		chunkMode = string(mode)
	}
	if len(args) >= 4 {
		env, ok := args[3].(*rt.Table)
		if !ok {
			return nil, errors.New("load: env must be a table")
		}
		chunkEnv = env
	}
	// TODO: use those
	_, _ = chunkName, chunkMode
	clos, err := rt.CompileLuaChunk(chunk, chunkEnv)
	if err != nil {
		return nil, err
	}
	next.Push(clos)
	return next, nil
}

func loadfile(t *rt.Thread, args []rt.Value, next rt.Cont) (rt.Cont, error) {
	chunk, chunkName, err := loadChunk(args)
	if err != nil {
		return nil, fmt.Errorf("loadfile: %s", err)
	}
	var chunkMode string
	var chunkEnv *rt.Table
	if len(args) >= 2 {
		mode, ok := args[1].(rt.String)
		if !ok {
			return nil, errors.New("loadfile: mode must be a string")
		}
		chunkMode = string(mode)
	}
	if len(args) >= 3 {
		env, ok := args[2].(*rt.Table)
		if !ok {
			return nil, errors.New("loadfile: env must be a table")
		}
		chunkEnv = env
	} else {
		chunkEnv = t.GlobalEnv()
	}
	// TODO: use name and mode
	_, _ = chunkName, chunkMode
	clos, err := rt.CompileLuaChunk(chunk, chunkEnv)
	if err != nil {
		return nil, err
	}
	next.Push(clos)
	return next, nil
}

func loadChunk(args []rt.Value) (chunk []byte, chunkName string, err error) {
	if len(args) == 0 {
		chunkName = "stdin"
		chunk, err = ioutil.ReadAll(os.Stdin)
	} else {
		path, ok := args[0].(rt.String)
		if !ok {
			err = errors.New("argument must be a string")
			return
		}
		chunk, err = ioutil.ReadFile(string(path))
		chunkName = string(path)
	}
	if err != nil {
		err = fmt.Errorf("error reading file: %s", err)
	}
	return
}

func dofile(t *rt.Thread, args []rt.Value, next rt.Cont) (rt.Cont, error) {
	chunk, chunkName, err := loadChunk(args)
	if err != nil {
		return nil, fmt.Errorf("dofile: %s", err)
	}
	// TODO: use chunkName
	_ = chunkName
	clos, err := rt.CompileLuaChunk(chunk, t.GlobalEnv())
	if err != nil {
		return nil, err
	}
	return rt.ContWithArgs(clos, nil, next), nil
}

func tonumber(t *rt.Thread, args []rt.Value, next rt.Cont) (rt.Cont, error) {
	if len(args) == 0 {
		return nil, errors.New("tonumber requires 1 argument")
	}
	n := args[0]
	if len(args) == 1 {
		n, _ = rt.ToNumber(n)
		next.Push(n)
		return next, nil
	}
	base, ok := args[1].(rt.Int)
	if !ok {
		return nil, errors.New("tonumber: base must be an integer")
	}
	if base < 2 || base > 36 {
		return nil, errors.New("tonumber: base out of range")
	}
	s, ok := n.(rt.String)
	if !ok {
		return nil, errors.New("tonunmber: argument 1 must be a string")
	}
	digits := bytes.Trim([]byte(s), " ")
	if len(digits) == 0 {
		return next, nil
	}
	var number rt.Int
	var sign rt.Int = 1
	if digits[0] == '-' {
		sign = -1
		digits = digits[1:]
		if len(digits) == 0 {
			return next, nil
		}
	}
	for _, digit := range digits {
		var d rt.Int
		switch {
		case '0' <= digit && digit <= '9':
			d = rt.Int(digit - '0')
		case 'a' <= digit && digit <= 'z':
			d = rt.Int(digit - 'a' + 10)
		case 'A' <= digit && digit <= 'Z':
			d = rt.Int(digit - 'A' + 10)
		default:
			return next, nil
		}
		if d >= base {
			return next, nil
		}
		number = number*base + d
	}
	next.Push(sign * number)
	return next, nil
}
