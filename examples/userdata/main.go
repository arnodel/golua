package main

import (
	"os"

	"github.com/arnodel/golua/examples/userdata/regexlib"
	"github.com/arnodel/golua/lib/base"
	"github.com/arnodel/golua/lib/packagelib"
	rt "github.com/arnodel/golua/runtime"
)

const code = `
regex = require("regex")
ptn = regex.new("[0-9]+")
print("ptn:", ptn)
match = ptn:find("hello there 123 yippee")
print("found:", match)
`

func main() {
	r := rt.New(os.Stdout)
	base.Load(r)
	packagelib.LibLoader.Run(r)
	regexlib.LibLoader.Run(r)
	chunk, _ := rt.CompileAndLoadLuaChunk("test", []byte(code), r.GlobalEnv())
	_, _ = rt.Call1(r.MainThread(), chunk)
}
