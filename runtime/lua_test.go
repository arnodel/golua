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
	rt.SolemnlyDeclareCompliance(rt.ComplyCpuSafe, r.SetEnvGoFunc(r.GlobalEnv(), "testudata", testudata, 1, false))
	return lib.LoadAll(r)
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
