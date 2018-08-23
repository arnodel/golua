package luatesting_test

import (
	"testing"

	"github.com/arnodel/golua/lib"
	"github.com/arnodel/golua/luatesting"
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
	err := luatesting.RunLuaTest([]byte(src), lib.Load)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLua(t *testing.T) {
	luatesting.RunLuaTestsInDir(t, "lua", lib.Load)
}
