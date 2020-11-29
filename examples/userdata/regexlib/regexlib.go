// Package regexlib is an example of how to make a go library for lua.  It
// allows using Go regular expressions in Lua code.  To use in a runtime r, add
// the following Go code:
//    regexlib.LibLoader.Run(r)
// Then in Lua code e.g.
//    regex = require"regex"
//    ptn = regex.new("[0-9]+")
//    match = ptn:find("hello there 123 yippee")
package regexlib

import (
	"fmt"
	"regexp"

	"github.com/arnodel/golua/lib/packagelib"
	rt "github.com/arnodel/golua/runtime"
)

var regexMeta *rt.Table

// LibLoader defines the name of the package and how to load it. Given a runtime
// r, call:
//    regexlib.LibLoader.Run(r)
// To load the package into the runtime (note that packagelib needs to be loaded
// first).
var LibLoader = packagelib.Loader{
	Load: load,
	Name: "regex",
}

// This function is the Load function of the LibLoader defined above.  It sets
// up a package (which is a lua table and returns it).
func load(r *rt.Runtime) rt.Value {
	// Make a new table
	pkg := rt.NewTable()

	// Add the "new" function to it
	rt.SetEnvGoFunc(pkg, "new", newRegex, 1, false)

	// Return the package table
	return pkg
}

// Creates a new regex userdata from a string.
func newRegex(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var s rt.String
	err := c.Check1Arg()
	if err == nil {
		s, err = c.StringArg(0)
	}
	if err != nil {
		return nil, err.AddContext(c)
	}
	re, compErr := regexp.Compile(string(s))
	if compErr != nil {
		return nil, rt.NewErrorE(compErr).AddContext(c)
	}
	return c.PushingNext(rt.NewUserData(re, regexMeta)), nil
}

// Hepler function that turns a Lua value to a Go regexp.
func valueToRegex(v rt.Value) (re *regexp.Regexp, ok bool) {
	var u *rt.UserData
	u, ok = v.(*rt.UserData)
	if ok {
		re, ok = u.Value().(*regexp.Regexp)
	}
	return
}

// Helper function that extracts a regexp arg from a continuation.
func regexArg(c *rt.GoCont, n int) (*regexp.Regexp, *rt.Error) {
	re, ok := valueToRegex(c.Arg(n))
	if ok {
		return re, nil
	}
	return nil, rt.NewErrorF("#%d must be a regex", n+1)
}

// This implements the 'find' method of a regexp.
func regexFind(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var re *regexp.Regexp
	var s rt.String

	// Check there are two arguments.
	err := c.CheckNArgs(2)
	if err == nil {
		// Ok, then get the first argument as a regexp.
		re, err = regexArg(c, 0)
	}
	if err == nil {
		// Ok, then get the second argument as a string.
		s, err = c.StringArg(1)
	}
	if err != nil {
		// Fail if an error occurred above
		return nil, err.AddContext(c)
	}
	// Find the pattern in the string and return it.
	match := re.FindString(string(s))
	return c.PushingNext(rt.String(match)), nil
}

// Implementation of the regex's '__tostring' metamethod.
func regexToString(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var re *regexp.Regexp
	err := c.Check1Arg()
	if err == nil {
		re, err = regexArg(c, 0)
	}
	if err != nil {
		return nil, err.AddContext(c)
	}
	s := rt.String(fmt.Sprintf("regex(%q)", re.String()))
	return c.PushingNext(s), nil
}

func init() {
	// At startup, build the metatable for the regex userdata.
	// First build a table of methods.
	regexMethods := rt.NewTable()
	rt.SetEnvGoFunc(regexMethods, "find", regexFind, 2, false)

	// Build the metatable
	regexMeta = rt.NewTable()
	rt.SetEnv(regexMeta, "__index", regexMethods)
	rt.SetEnvGoFunc(regexMeta, "__tostring", regexToString, 1, false)
}
