package runtime

import "sync/atomic"

// floatFormat holds the function set by SetFloatFormat, or nil.
var floatFormat atomic.Pointer[func(float64) string]

// SetFloatFormat makes f convert floats to strings wherever Lua converts a
// number to a string (tostring, print, concatenation, string arguments,
// table.concat, io.write, string.format's %s), in place of FloatToString.
// Passing nil restores FloatToString.
//
// The manual leaves this format unspecified (§3.4.3), and Lua versions
// differ: for example 5.3 and 5.4 print 14 significant digits and 5.5 prints
// as many as the float needs.  SetFloatFormat lets a program reproduce the
// output of the version it replaces.  It applies to every Runtime in the
// program; call it before running Lua code.
//
// string.format's %q is not affected: it always uses FloatToString, so that
// its result reads back as the same float.
func SetFloatFormat(f func(float64) string) {
	if f == nil {
		floatFormat.Store(nil)
		return
	}
	floatFormat.Store(&f)
}

// floatToString converts x as SetFloatFormat says.
func floatToString(x float64) string {
	if f := floatFormat.Load(); f != nil {
		return (*f)(x)
	}
	return FloatToString(x)
}
