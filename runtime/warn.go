package runtime

import (
	"fmt"
	"io"
	"strings"
)

// Warner is the interface for the Lua warning system.  The Warn function emits
// warning, when called with one argument, if the argument starts with '@' then
// the message is a control message.  These messages are not emitted but control
// the behaviour of the Warner instance.
type Warner interface {
	Warn(msgs ...string)
}

// The default Warner type.  It logs messages to a given io.Writer.  Note that
// it is off by default.  Issue Warn("@on") to turn it on.
type LogWarner struct {
	on   bool
	dest io.Writer
	pfx  string
}

var _ Warner = (*LogWarner)(nil)

// NewLogWarner returns a new LogWarner that will write to dest with the given
// prefix.
func NewLogWarner(dest io.Writer, pfx string) *LogWarner {
	return &LogWarner{dest: dest, pfx: pfx}
}

// Warn concatenates its arguments and emits a warning message (written to
// dest).  It understands two control messages: "@on" and "@off", with the
// obvious meaning.
func (w *LogWarner) Warn(msgs ...string) {
	if len(msgs) == 1 && len(msgs[0]) > 0 && msgs[0][0] == '@' {
		// Control message
		switch msgs[0] {
		case "@on":
			w.on = true
		case "@off":
			w.on = false
		}
		return
	}
	if w.on {
		fmt.Fprintf(w.dest, "%s%s\n", w.pfx, strings.Join(msgs, ""))
	}
}
