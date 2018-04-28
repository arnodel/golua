package luatests

import (
	"bytes"
	"fmt"
	"io"

	"github.com/arnodel/golua/lib/base"
	"github.com/arnodel/golua/lib/coroutine"
	"github.com/arnodel/golua/runtime"
)

func RunSource(source []byte, output io.Writer) {
	r := runtime.New(output)
	base.Load(r)
	coroutine.Load(r)
	t := r.MainThread()
	clos, err := runtime.CompileLuaChunk(source, r.GlobalEnv())
	err = runtime.Call(t, clos, nil, runtime.NewTerminationWith(0, false))
	if err != nil {
		fmt.Fprintf(output, "!!! runtime: %s", err)
	}
}

func RunLuaTest(source []byte) error {
	outputBuf := new(bytes.Buffer)
	checkers := ExtractLineCheckers(source)
	RunSource(source, outputBuf)
	return CheckLines(outputBuf.Bytes(), checkers)
}
