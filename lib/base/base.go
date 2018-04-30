package base

import (
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
	rt.SetEnvGoFunc(env, "dofile", dofile, 1, false)
	rt.SetEnvGoFunc(env, "error", errorF, 2, false)
	rt.SetEnv(env, "_G", env)
	rt.SetEnvGoFunc(env, "getmetatable", getmetatable, 1, false)
	// TODO: ipairs
	rt.SetEnvGoFunc(env, "load", load, 4, false)
	rt.SetEnvGoFunc(env, "loadfile", loadfile, 3, false)
	// TODO: next
	// TODO: pairs
	rt.SetEnvGoFunc(env, "pcall", pcall, 1, true)
	rt.SetEnvGoFunc(env, "print", print, 0, true)
	rt.SetEnvGoFunc(env, "rawequal", rawequal, 2, false)
	rt.SetEnvGoFunc(env, "rawget", rawget, 2, false)
	rt.SetEnvGoFunc(env, "rawlen", rawlen, 1, false)
	rt.SetEnvGoFunc(env, "rawset", rawset, 3, false)
	rt.SetEnvGoFunc(env, "select", selectF, 1, true)
	rt.SetEnvGoFunc(env, "setmetatable", setmetatable, 2, false)
	rt.SetEnvGoFunc(env, "tonumber", tonumber, 2, false)
	rt.SetEnvFunc(env, "tostring", tostring)
	rt.SetEnvGoFunc(env, "type", typeString, 1, false)
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
