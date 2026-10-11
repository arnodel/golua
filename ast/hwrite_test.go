package ast

import (
	"strings"
	"testing"
)

func hwrite(n interface{ HWrite(HWriter) }) string {
	var b strings.Builder
	n.HWrite(NewIndentWriter(&b))
	return b.String()
}

// HWrite renders names as their text, not as the structs that hold them.
func TestHWriteRendersNames(t *testing.T) {
	name := func(s string) Name { return Name{Val: s} }
	for _, tc := range []struct {
		node interface{ HWrite(HWriter) }
		want string
	}{
		{ForStat{Var: name("i")}, "for i"},
		{GotoStat{Label: name("done")}, "goto done"},
		{BFunctionCall{Target: name("obj"), Method: name("m")}, "method: m"},
		{LocalStat{NameAttribs: []NameAttrib{{Name: name("x")}}}, "name_0: x"},
		{LocalStat{NameAttribs: []NameAttrib{{Name: name("c"), Attrib: &DeclAttrib{Type: ConstAttrib}}}}, "name_0: c <const>"},
		{LocalStat{NameAttribs: []NameAttrib{{Name: name("f"), Attrib: &DeclAttrib{Type: CloseAttrib}}}}, "name_0: f <close>"},
		{GlobalStat{NameAttribs: []NameAttrib{{Name: name("g")}}}, "name_0: g"},
		{name("a%sb"), "a%sb"},
		{Function{ParList: ParList{Params: []Name{name("p%d"), name("q")}, HasDots: true}}, "(p%d, q, ...)"},
	} {
		got := hwrite(tc.node)
		if !strings.Contains(got, tc.want) {
			t.Errorf("HWrite = %q, want it to contain %q", got, tc.want)
		}
		if strings.Contains(got, "{") || strings.Contains(got, "%!") {
			t.Errorf("HWrite = %q: a struct or a format error leaked into the output", got)
		}
	}
}
