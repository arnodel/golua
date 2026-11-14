package runtime

import (
	"fmt"
	"strings"
)

// DebugInfo contains info about a continuation that can be looked at for
// debugging purposes (and tracebacks).
type DebugInfo struct {
	Source      string
	Name        string
	CurrentLine int32
}

// String formats the data contained in DebugInfo in a human-readable way.
func (i DebugInfo) String() string {
	return fmt.Sprintf("file=%s func=%s line=%d", i.Source, i.Name, i.CurrentLine)
}

// maxShortSource is the length to which shortSource cuts the names it
// shortens.
const maxShortSource = 59

// shortSource returns the name of a chunk as an error message or a traceback
// shows it.  The first character of the name a Lua program gives a chunk can
// be a mark that says what the name is, and is not shown:
//
//   - "=name" is a name to show as it is, cut to its first 59 bytes;
//   - "@name" is the name of a file; one longer than 59 bytes is shown as
//     "..." and the end of the name, the part that tells files apart.
//
// Any other name is shown whole.
func shortSource(source string) string {
	switch {
	case strings.HasPrefix(source, "="):
		name := source[1:]
		if len(name) > maxShortSource {
			name = name[:maxShortSource]
		}
		return name
	case strings.HasPrefix(source, "@"):
		name := source[1:]
		if len(name) > maxShortSource {
			const dots = "..."
			name = dots + name[len(name)-(maxShortSource-len(dots)):]
		}
		return name
	default:
		return source
	}
}
