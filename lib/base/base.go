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
	rt.SetEnvGoFunc(env, "assert", assert, 2, false)
	rt.SetEnvGoFunc(env, "dofile", dofile, 1, false)
	rt.SetEnvGoFunc(env, "error", errorF, 2, false)
	rt.SetEnv(env, "_G", env)
	rt.SetEnvGoFunc(env, "getmetatable", getmetatable, 1, false)
	rt.SetEnvGoFunc(env, "ipairs", ipairs, 1, false)
	rt.SetEnvGoFunc(env, "load", load, 4, false)
	rt.SetEnvGoFunc(env, "loadfile", loadfile, 3, false)
	rt.SetEnv(env, "next", nextGoFunc)
	rt.SetEnvGoFunc(env, "pairs", pairs, 1, false)
	rt.SetEnvGoFunc(env, "pcall", pcall, 1, true)
	rt.SetEnvGoFunc(env, "print", print, 0, true)
	rt.SetEnvGoFunc(env, "rawequal", rawequal, 2, false)
	rt.SetEnvGoFunc(env, "rawget", rawget, 2, false)
	rt.SetEnvGoFunc(env, "rawlen", rawlen, 1, false)
	rt.SetEnvGoFunc(env, "rawset", rawset, 3, false)
	rt.SetEnvGoFunc(env, "select", selectF, 1, true)
	rt.SetEnvGoFunc(env, "setmetatable", setmetatable, 2, false)
	rt.SetEnvGoFunc(env, "tonumber", tonumber, 2, false)
	rt.SetEnvGoFunc(env, "tostring", tostring, 1, false)
	rt.SetEnvGoFunc(env, "type", typeString, 1, false)
	rt.SetEnv(env, "_VERSION", rt.String("Golua 5.3"))
	// TODO: xpcall
}

func toString(t *rt.Thread, v rt.Value) (rt.String, *rt.Error) {
	next := rt.NewTerminationWith(1, false)
	err, ok := rt.Metacall(t, v, "__tostring", []rt.Value{v}, next)
	if err != nil {
		return "", err
	}
	if ok {
		v = next.Get(0)
	}
	s, ok := rt.AsString(v)
	if !ok {
		s = rt.String(fmt.Sprintf("%s: <addr>", rt.Type(v)))
	}
	return s, nil
}

func tostring(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if c.NArgs() == 0 {
		return nil, rt.NewErrorS("1 argument required")
	}
	s, err := toString(t, c.Arg(0))
	if err != nil {
		return nil, err.AddContext(c)
	}
	c.Next().Push(s)
	return c.Next(), nil
}

func loadChunk(args []rt.Value) (chunk []byte, chunkName string, err error) {
	if len(args) == 0 {
		chunkName = "stdin"
		chunk, err = ioutil.ReadAll(os.Stdin)
	} else {
		path, ok := args[0].(rt.String)
		if !ok {
			err = errors.New("#1 must be a string")
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
