package luatesting_test

import (
	"runtime"
	"testing"

	"github.com/arnodel/golua/lib"
	"github.com/arnodel/golua/luatesting"

	rt "github.com/arnodel/golua/runtime"
)

func TestRunLuaTest(t *testing.T) {
	src := `
print("hello, world!")
--> =hello, world!

print(1+2)
--> =3

print(1 == 1.0)
--> =true

error("hello")
--> ~!!! runtime:.*
`
	err := luatesting.RunLuaTest([]byte(src), lib.LoadAll)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLua(t *testing.T) {
	luatesting.RunLuaTestsInDir(t, "lua", setup)
}

func setup(r *rt.Runtime) func() {
	cleanup := lib.LoadAll(r)
	g := r.GlobalEnv()
	r.SetEnv(g, "goos", rt.StringValue(runtime.GOOS))
	return cleanup
}
