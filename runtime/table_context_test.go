package runtime

import (
	"io"
	"testing"
)

func TestRuntimeNewTableUnrestricted(t *testing.T) {
	r := New(io.Discard)
	defer r.Close(nil)
	table := r.NewTable()
	r.SetTable(table, StringValue("answer"), IntValue(42))
	if table.Get(StringValue("answer")) != IntValue(42) || r.UsedResources().Memory != 0 {
		t.Fatal("unrestricted table behavior changed")
	}
}
