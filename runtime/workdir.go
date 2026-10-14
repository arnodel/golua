package runtime

import "path/filepath"

// resolveWorkDir returns the working directory of a new context, given its
// parent's working directory and the one requested in its RuntimeContextDef.
// An empty working directory means the process's working directory.
func resolveWorkDir(parent, dir string) string {
	if dir == "" {
		return parent
	}
	if filepath.IsAbs(dir) {
		return filepath.Clean(dir)
	}
	if parent != "" {
		return filepath.Join(parent, dir)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		// The process's working directory is unknown, so leave dir relative
		// to it.
		return filepath.Clean(dir)
	}
	return abs
}
