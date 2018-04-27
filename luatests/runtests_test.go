package luatests

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
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
	err := RunLuaTest([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
}

func TestLua(t *testing.T) {
	runTest := func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) != ".lua" {
			return nil
		}
		t.Run(path, func(t *testing.T) {
			src, err := ioutil.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			err = RunLuaTest(src)
			if err != nil {
				t.Fatal(err)
			}
		})
		return nil
	}
	filepath.Walk("lua", runTest)
}
