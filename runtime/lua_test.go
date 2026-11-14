package runtime_test

import (
	"fmt"
	"io"
	"testing"

	"github.com/arnodel/golua/lib"
	"github.com/arnodel/golua/luatesting"
	rt "github.com/arnodel/golua/runtime"
)

func TestRuntime(t *testing.T) {
	luatesting.RunLuaTestsInDir(t, "lua", setup)
}

func setup(r *rt.Runtime) func() {
	env := r.GlobalEnv()
	rt.SolemnlyDeclareCompliance(rt.ComplyCpuSafe, r.SetEnvGoFunc(env, "testudata", testudata, 1, false))
	r.SetEnvGoFunc(env, "testtailcall", testtailcall, 1, true)
	r.SetEnvGoFunc(env, "testcatch", testcatch, 1, true)
	return lib.LoadAll(r)
}

// testtailcall(f, ...) returns f(...) as a Go function does that calls a
// function last: it does not wait for it to return.
func testtailcall(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	f, err := c.CallableArg(0)
	if err != nil {
		return nil, err
	}
	cont := f.Continuation(t, c.Next())
	cont.PushEtc(t.Runtime, c.Etc())
	return cont, nil
}

// testcatch(f, ...) calls f(...) and returns the message of its error if
// there is one, as a Go function does that keeps every error to itself.
func testcatch(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	res := rt.NewTerminationWith(c, 0, false)
	if err := rt.Call(t, c.Arg(0), c.Etc(), res); err != nil {
		return c.PushingNext1(t.Runtime, rt.StringValue(err.Error())), nil
	}
	return c.Next(), nil
}

// This return a test userdata that prints a message when released.  Allows testing
func testudata(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	arg, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	return c.PushingNext1(t.Runtime, t.NewUserDataValue(&TestUData{val: arg, stdout: t.Stdout}, nil)), nil
}

type TestUData struct {
	val    string
	stdout io.Writer
}

func (t *TestUData) ReleaseResources(d *rt.UserData) {
	fmt.Fprintf(t.stdout, "**release %s**\n", t.val)
}
