package base

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	rt "github.com/arnodel/golua/runtime"
)

func Load(r *rt.Runtime) {
	env := r.GlobalEnv()
	r.SetEnvGoFunc(env, "assert", assert, 1, true)
	r.SetEnvGoFunc(env, "collectgarbage", collectgarbage, 2, false)
	r.SetEnvGoFunc(env, "dofile", dofile, 1, false)
	r.SetEnvGoFunc(env, "error", errorF, 2, false)
	r.SetEnv(env, "_G", rt.TableValue(env))
	r.SetEnvGoFunc(env, "getmetatable", getmetatable, 1, false)
	r.SetEnvGoFunc(env, "ipairs", ipairs, 1, false)
	r.SetEnvGoFunc(env, "load", load, 4, false)
	r.SetEnvGoFunc(env, "loadfile", loadfile, 3, false)
	r.SetEnv(env, "next", rt.FunctionValue(nextGoFunc))
	r.SetEnvGoFunc(env, "pairs", pairs, 1, false)
	r.SetEnvGoFunc(env, "pcall", pcall, 1, true)
	r.SetEnvGoFunc(env, "print", print, 0, true)
	r.SetEnvGoFunc(env, "rawequal", rawequal, 2, false)
	r.SetEnvGoFunc(env, "rawget", rawget, 2, false)
	r.SetEnvGoFunc(env, "rawlen", rawlen, 1, false)
	r.SetEnvGoFunc(env, "rawset", rawset, 3, false)
	r.SetEnvGoFunc(env, "select", selectF, 1, true)
	r.SetEnvGoFunc(env, "setmetatable", setmetatable, 2, false)
	r.SetEnvGoFunc(env, "tonumber", tonumber, 2, false)
	r.SetEnvGoFunc(env, "tostring", tostring, 1, false)
	r.SetEnvGoFunc(env, "type", typeString, 1, false)
	r.SetEnv(env, "_VERSION", rt.StringValue("Golua 5.3"))
	// TODO: xpcall
}

func ToString(t *rt.Thread, v rt.Value) (string, *rt.Error) {
	next := rt.NewTerminationWith(1, false)
	err, ok := rt.Metacall(t, v, "__tostring", []rt.Value{v}, next)
	if err != nil {
		return "", err
	}
	if ok {
		s, ok := rt.ToString(next.Get(0))
		if !ok {
			return "", rt.NewErrorS("'__tostring' must return a string")
		}
		return s, nil
	}
	s, ok := rt.ToString(v)
	// TODO: fix this hack
	if s == "" && !ok {
		s = fmt.Sprintf("%s: <addr>", rt.Type(v))
	}
	return s, nil
}

func tostring(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	s, err := ToString(t, c.Arg(0))
	if err != nil {
		return nil, err.AddContext(c)
	}
	return c.PushingNext(t.Runtime, rt.StringValue(s)), nil
}

func loadChunk(t *rt.Thread, args []rt.Value) (chunk []byte, chunkName string, err error) {
	budget := t.LinearUnused(10)
	var reader io.Reader
	if len(args) == 0 {
		chunkName = "stdin"
		reader = os.Stdin
	} else {
		var ok bool
		chunkName, ok = args[0].TryString()
		if !ok {
			return nil, chunkName, errors.New("#1 must be a string")
		}
		f, err := os.Open(chunkName)
		if err != nil {
			return nil, chunkName, fmt.Errorf("error opeing file: %s", err)
		}
		defer f.Close()
		reader = f
	}
	if budget > 0 {
		reader = io.LimitReader(reader, int64(budget))
	}
	chunk, err = ioutil.ReadAll(reader)
	if err != nil {
		return nil, chunkName, fmt.Errorf("error reading file: %s", err)
	}
	t.LinearRequire(10, uint64(len(chunk)))
	return chunk, chunkName, nil
}
