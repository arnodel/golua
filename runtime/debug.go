package runtime

// DebugInfo contains info about a continuation that can be looked at for
// debuggin purposes (and tracebacks)
type DebugInfo struct {
	Source      string
	Name        string
	CurrentLine int32
}
