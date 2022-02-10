package runtime

import "fmt"

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
