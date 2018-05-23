package luatests

import (
	"bytes"
	"fmt"
	"io"

	"github.com/arnodel/golua/lib/base"
	"github.com/arnodel/golua/lib/coroutine"
	"github.com/arnodel/golua/lib/packagelib"
	"github.com/arnodel/golua/lib/stringlib"
	"github.com/arnodel/golua/runtime"
)

func RunSource(source []byte, output io.Writer) {
	r := runtime.New(output)
	base.Load(r)
	coroutine.Load(r)
	packagelib.Load(r)
	stringlib.Load(r)
	t := r.MainThread()
	// TODO: use the file name
	clos, err := runtime.CompileAndLoadLuaChunk("luatest", source, r.GlobalEnv())
	if err != nil {
		fmt.Fprintf(output, "!!! parsing: %s", err)
		return
	}
	cerr := runtime.Call(t, clos, nil, runtime.NewTerminationWith(0, false))
	if cerr != nil {
		fmt.Fprintf(output, "!!! runtime: %s", cerr)
	}
}

func RunLuaTest(source []byte) error {
	outputBuf := new(bytes.Buffer)
	checkers := ExtractLineCheckers(source)
	RunSource(source, outputBuf)
	return CheckLines(outputBuf.Bytes(), checkers)
}
