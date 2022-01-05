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

var regexMetaKey = rt.StringValue("regexMeta")

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
func load(r *rt.Runtime) (rt.Value, func()) {
	// First build a table of methods.
	regexMethods := rt.NewTable()
	r.SetEnvGoFunc(regexMethods, "find", regexFind, 2, false)

	// Build the metatable
	regexMeta := rt.NewTable()
	r.SetEnv(regexMeta, "__index", rt.TableValue(regexMethods))
	r.SetEnvGoFunc(regexMeta, "__tostring", regexToString, 1, false)
	r.SetRegistry(regexMetaKey, rt.TableValue(regexMeta))

	// Make a new table
	pkg := rt.NewTable()

	// Add the "new" function to it
	r.SetEnvGoFunc(pkg, "new", newRegex, 1, false)

	// Return the package table
	return rt.TableValue(pkg), nil
}

// Creates a new regex userdata from a string.
func newRegex(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var s string
	err := c.Check1Arg()
	if err == nil {
		s, err = c.StringArg(0)
	}
	if err != nil {
		return nil, err
	}
	re, compErr := regexp.Compile(string(s))
	if compErr != nil {
		return nil, rt.NewErrorE(compErr)
	}
	regexMeta := t.Registry(regexMetaKey)
	return c.PushingNext(t.Runtime, rt.UserDataValue(rt.NewUserData(re, regexMeta.AsTable()))), nil
}

// Hepler function that turns a Lua value to a Go regexp.
func valueToRegex(v rt.Value) (re *regexp.Regexp, ok bool) {
	var u *rt.UserData
	u, ok = v.TryUserData()
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
	var (
		re *regexp.Regexp
		s  string
	)
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
		return nil, err
	}
	// Find the pattern in the string and return it.
	match := re.FindString(string(s))
	return c.PushingNext(t.Runtime, rt.StringValue(match)), nil
}

// Implementation of the regex's '__tostring' metamethod.
func regexToString(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var re *regexp.Regexp
	err := c.Check1Arg()
	if err == nil {
		re, err = regexArg(c, 0)
	}
	if err != nil {
		return nil, err
	}
	s := rt.StringValue(fmt.Sprintf("regex(%q)", re.String()))
	return c.PushingNext(t.Runtime, s), nil
}
