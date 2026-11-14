package runtime

import (
	"strings"
	"testing"
)

func TestShortSource(t *testing.T) {
	long := strings.Repeat("a", 100) + "xyz"
	tests := []struct{ source, want string }{
		// "=" marks a name to show as it is.
		{"=name", "name"},
		{"=", ""},
		{"==name", "=name"},
		{"=two\nlines", "two\nlines"},
		{"=" + long[:59], long[:59]},
		{"=" + long[:60], long[:59]},
		{"=" + long, long[:59]},
		// "@" marks the name of a file, whose end is the part to keep.
		{"@dir/file.lua", "dir/file.lua"},
		{"@", ""},
		{"@" + long[44:], long[44:]},
		{"@" + long[43:], "..." + long[47:]},
		{"@" + long, "..." + long[47:]},
		// Any other name is shown whole.
		{"", ""},
		{"name", "name"},
		{"a=b", "a=b"},
		{"user@host", "user@host"},
		{long, long},
	}
	for _, test := range tests {
		got := shortSource(test.source)
		if got != test.want {
			t.Errorf("shortSource(%q) = %q, want %q", test.source, got, test.want)
		}
		if marked := strings.HasPrefix(test.source, "=") || strings.HasPrefix(test.source, "@"); marked && len(got) > maxShortSource {
			t.Errorf("shortSource(%q) is %d bytes long", test.source, len(got))
		}
	}
}
