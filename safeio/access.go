package safeio

import (
	"errors"
	"fmt"
	"math/bits"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FSAction is a set of actions that can be performed on files.
type FSAction uint8

const (
	ReadAction   FSAction = 1 << iota // Read the contents of a file
	WriteAction                       // Write to an existing file
	CreateAction                      // Create a new file
	DeleteAction                      // Delete or rename a file

	NoActions  FSAction = 0
	AllActions          = ReadAction | WriteAction | CreateAction | DeleteAction
)

// FSGrant gives permission to perform some actions on files in a directory
// (recursively).
type FSGrant struct {
	Dir     string
	Actions FSAction
}

// FSAccess describes which files can be accessed and what actions can be
// performed on them.  It is either unrestricted (the Unrestricted value) or a
// set of directories, each opened as an *os.Root so that accessing a file can
// never escape the directory (e.g. via ".." or symbolic links).
//
// An action on a file is allowed if any directory containing the file is
// granted all the actions required.  This means that grants cannot be used to
// restrict access to a subdirectory of a granted directory: if a file can be
// reached via several grants, the most permissive one applies.
//
// An FSAccess value is immutable apart from being closed.
type FSAccess struct {
	parent       *FSAccess
	unrestricted bool
	roots        []fsRoot // sorted from longest to shortest dir
}

type fsRoot struct {
	dir     string // absolute and clean
	root    *os.Root
	actions FSAction
}

var (
	// Unrestricted allows any action on any file.
	Unrestricted = &FSAccess{unrestricted: true}

	// NoAccess forbids any action on any file.
	NoAccess = &FSAccess{}
)

// ErrNotAllowed is returned when an action is not allowed by the FSAccess.
var ErrNotAllowed = errors.New("safeio: operation not allowed")

// FSAccessor is implemented by types that can tell what file system access is
// currently allowed (e.g. *runtime.Runtime).
type FSAccessor interface {
	FSAccess() *FSAccess
}

// IsUnrestricted returns true if a allows any action on any file.
func (a *FSAccess) IsUnrestricted() bool {
	return a != nil && a.unrestricted
}

// Parent returns the FSAccess that a was derived from via Restrict.
func (a *FSAccess) Parent() *FSAccess {
	if a == nil {
		return nil
	}
	return a.parent
}

// Restrict returns a new FSAccess allowing only the given grants.  The
// returned value can never allow more than a does:
//
//   - if a is unrestricted, each directory is opened as an *os.Root;
//   - otherwise each directory must be inside a directory granted by a, and it
//     is opened through that directory's *os.Root, with the actions narrowed to
//     those allowed by a.
//
// The caller is responsible for calling Close on the returned value when it is
// no longer needed.
func (a *FSAccess) Restrict(grants []FSGrant) (*FSAccess, error) {
	b := &FSAccess{parent: a}
	for _, g := range grants {
		r, err := a.openRoot(g)
		if err != nil {
			b.Close()
			return nil, fmt.Errorf("cannot grant access to %q: %w", g.Dir, err)
		}
		b.roots = append(b.roots, r)
	}
	sort.SliceStable(b.roots, func(i, j int) bool {
		return len(b.roots[i].dir) > len(b.roots[j].dir)
	})
	return b, nil
}

// Close releases the directories opened by a (not those of its parent).  Files
// opened via a remain open.
func (a *FSAccess) Close() error {
	if a == nil {
		return nil
	}
	var err error
	for _, r := range a.roots {
		err = errors.Join(err, r.root.Close())
	}
	return err
}

func (a *FSAccess) openRoot(g FSGrant) (fsRoot, error) {
	dir, err := filepath.Abs(g.Dir)
	if err != nil {
		return fsRoot{}, err
	}
	if a.IsUnrestricted() {
		root, err := os.OpenRoot(dir)
		if err != nil {
			return fsRoot{}, err
		}
		return fsRoot{dir: dir, root: root, actions: g.Actions}, nil
	}
	// Open the directory through the parent directory that allows the most of
	// the requested actions.
	var (
		parent fsRoot
		rel    string
		found  bool
	)
	for _, r := range a.rootsContaining(dir) {
		if !found || bits.OnesCount8(uint8(r.actions&g.Actions)) > bits.OnesCount8(uint8(parent.actions&g.Actions)) {
			parent, rel, found = r.fsRoot, r.rel, true
		}
	}
	if !found {
		return fsRoot{}, ErrNotAllowed
	}
	root, err := parent.root.OpenRoot(rel)
	if err != nil {
		return fsRoot{}, err
	}
	return fsRoot{dir: dir, root: root, actions: g.Actions & parent.actions}, nil
}

type rootMatch struct {
	fsRoot
	rel string
}

// rootsContaining returns the roots whose directory lexically contains the
// absolute path, with the path relative to each of them, longest directory
// first.
func (a *FSAccess) rootsContaining(path string) []rootMatch {
	if a == nil {
		return nil
	}
	var matches []rootMatch
	for _, r := range a.roots {
		rel, err := filepath.Rel(r.dir, path)
		if err != nil || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		matches = append(matches, rootMatch{fsRoot: r, rel: rel})
	}
	return matches
}

// resolve finds a granted directory containing the named file which allows
// all the given actions.  It returns the directory's *os.Root and the name of
// the file relative to it.  The lookup is lexical: the *os.Root then makes
// sure the file cannot be outside the directory.
func (a *FSAccess) resolve(name string, actions FSAction) (*os.Root, string, error) {
	path, err := filepath.Abs(name)
	if err != nil {
		return nil, "", err
	}
	for _, r := range a.rootsContaining(path) {
		if r.actions&actions == actions {
			return r.root, r.rel, nil
		}
	}
	return nil, "", ErrNotAllowed
}

// resolve2 is like resolve but for two files that must be in the same granted
// directory (e.g. for renaming).
func (a *FSAccess) resolve2(name1, name2 string, actions FSAction) (*os.Root, string, string, error) {
	path1, err := filepath.Abs(name1)
	if err != nil {
		return nil, "", "", err
	}
	path2, err := filepath.Abs(name2)
	if err != nil {
		return nil, "", "", err
	}
	for _, r := range a.rootsContaining(path1) {
		if r.actions&actions != actions {
			continue
		}
		for _, r2 := range a.rootsContaining(path2) {
			if r2.root == r.root {
				return r.root, r.rel, r2.rel, nil
			}
		}
	}
	return nil, "", "", ErrNotAllowed
}

// ParseFSActions turns a string such as "rw" into an FSAction.  Valid
// characters are 'r' (read), 'w' (write), 'c' (create), 'd' (delete) and '*'
// (all actions).
func ParseFSActions(s string) (FSAction, error) {
	var actions FSAction
	for _, c := range s {
		switch c {
		case 'r':
			actions |= ReadAction
		case 'w':
			actions |= WriteAction
		case 'c':
			actions |= CreateAction
		case 'd':
			actions |= DeleteAction
		case '*':
			actions |= AllActions
		default:
			return 0, fmt.Errorf("invalid file action %q (expected one of \"rwcd*\")", c)
		}
	}
	return actions, nil
}
