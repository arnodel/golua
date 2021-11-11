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
	r := rt.New(os.Stdout)      // Create runtime
	base.Load(r)                // Load base lib (needed for print)
	packagelib.LibLoader.Run(r) // Load package lib (needed for require)
	regexlib.LibLoader.Run(r)   // Load our example lib

	// Now compile and run the lua code
	chunk, _ := r.CompileAndLoadLuaChunk("test", []byte(code), r.GlobalEnv())
	_, _ = rt.Call1(r.MainThread(), rt.FunctionValue(chunk))
}
