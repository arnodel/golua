package code

import (
	"strings"
	"testing"
)

// A function compiled to a single opcode, such as function() end, has a span
// that starts and ends on the same opcode.  Disassembling it used to loop
// forever.
func TestDisassembleOneOpcodeFunction(t *testing.T) {
	ret := TailCall(ValueReg(0))
	unit := &Unit{
		Code:  []Opcode{ret, ret, ret, ret},
		Lines: make([]int32, 4),
		Constants: []Constant{
			Code{Name: "main", StartOffset: 0, EndOffset: 3},
			Code{Name: "f", StartOffset: 3, EndOffset: 4},
		},
	}
	var b strings.Builder
	unit.Disassemble(&b)
	var spans []string
	for _, line := range strings.Split(b.String(), "\n") {
		if strings.Contains(line, "tailcall") {
			spans = append(spans, strings.Fields(line)[1])
		}
	}
	want := []string{"main", "|", `\`, "f"}
	if strings.Join(spans, " ") != strings.Join(want, " ") {
		t.Errorf("spans %q, want %q\n%s", spans, want, b.String())
	}
}
