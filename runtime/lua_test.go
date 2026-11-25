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
	r.SetEnvGoFunc(env, "testrun", testrun, 1, true)
	return lib.LoadAll(r)
}

// testrun(f, ...) calls f(...) and waits for it to return, as a Go function
// does that makes the continuation of the call with Continue and runs it.
func testrun(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	cont, err := rt.Continue(t, c.Arg(0), rt.NewTerminationWith(c, 0, false))
	if err != nil {
		return nil, err
	}
	cont.PushEtc(t.Runtime, c.Etc())
	if err := t.RunContinuation(cont); err != nil {
		return nil, err
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
