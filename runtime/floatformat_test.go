package runtime_test

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/arnodel/golua/lib"
	rt "github.com/arnodel/golua/runtime"
)

// Every conversion of a float to a string goes through FloatToString, except
// string.format's %q.
func TestFloatToStringVar(t *testing.T) {
	rt.FloatToString = func(x float64) string { return "<" + strconv.FormatFloat(x, 'g', 14, 64) + ">" }
	t.Cleanup(func() { rt.FloatToString = rt.LuaFloatToString })

	var out bytes.Buffer
	r := rt.New(&out)
	defer lib.LoadAll(r)()
	src := `
		local x = 1/3
		print(x)
		print(tostring(x), "a" .. x, x .. "b")
		print(table.concat({x, 2.5, 3}, ","))
		print(string.format("%s", x), string.upper(x), #tostring(1e14))
		print(string.format("%q", 0.5), string.format("%q", 1e14))
		print(1, -0.0, 1 // 1)
	`
	chunk, err := r.CompileAndLoadLuaChunk("test", []byte(src), rt.TableValue(r.GlobalEnv()))
	if err != nil {
		t.Fatal(err)
	}
	th := r.MainThread()
	if err := rt.Call(th, rt.FunctionValue(chunk), nil, rt.NewTerminationWith(nil, 0, false)); err != nil {
		t.Fatal(err)
	}
	want := `<0.33333333333333>
<0.33333333333333>	a<0.33333333333333>	<0.33333333333333>b
<0.33333333333333>,<2.5>,3
<0.33333333333333>	<0.33333333333333>	7
0.5	100000000000000.0
1	<-0>	1
`
	if got := out.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}
