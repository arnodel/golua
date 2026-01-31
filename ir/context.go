package ir

import "fmt"

// GlobalDeclType represents the type of a global variable declaration
type GlobalDeclType uint8

const (
	MutableGlobal GlobalDeclType = 1 << iota
	ConstGlobal
	LegacyGlobal
	NoDeclaredGlobal = 0
)

func (d GlobalDeclType) IsLegacy() bool {
	return d&LegacyGlobal != 0
}

func (d GlobalDeclType) IsDeclared() bool {
	return d&LegacyGlobal == 0 && d != 0
}

func (d GlobalDeclType) IsAllowed() bool {
	return d != 0
}

func (d GlobalDeclType) StripLegacy() GlobalDeclType {
	return d &^ LegacyGlobal
}

type lexicalScope struct {
	reg    map[Name]taggedReg     // maps variable names to registers
	label  map[Name]labelWithLine // maps label names to labels
	height int                    // This is the height of the close stack in this scope

	// Global variable tracking for Lua 5.5
	globalDecls        map[Name]GlobalDeclType // tracks declared globals in this scope
	globalWildcardDecl GlobalDeclType          // tracks "global *" or "global<const> *"
}

func (s lexicalScope) getLabel(name Name) (label Label, line int, ok bool) {
	ll, ok := s.label[name]
	if ok {
		label = ll.Label
		line = ll.line
	}
	return
}

type taggedReg struct {
	reg  Register
	tags uint
}

type labelWithLine struct {
	Label
	line int
}

// A lexicalContext maintains nested mappings of names to registers and jump
// labels corresponding to nested lexical scopes.  It facilitates accessing the
// register of label associated with a given name at a given point in the code.
type lexicalContext []lexicalScope

// getRegister returns the register associated with the given name if it exists
// in one of the accessible lexical scopes.  Otherwise it sets ok to false.
// If an explicit global declaration for the name is found in a more nested scope
// than any local register, the local is considered shadowed and ok is false.
// TODO: explain tags.
func (c lexicalContext) getRegister(name Name, tags uint) (reg Register, ok bool) {
	for i := len(c) - 1; i >= 0; i-- {
		// Check for explicit global declaration first - it shadows outer locals
		if _, hasGlobal := c[i].globalDecls[name]; hasGlobal {
			return // Global declaration shadows any local in outer scopes
		}
		var tr taggedReg
		tr, ok = c[i].reg[name]
		if ok {
			reg = tr.reg
			if tags != 0 {
				tr.tags |= tags
				c[i].reg[name] = tr
			}
			break
		}
	}
	return
}

// getLabel returns the label associated with the given name if it has been
// defined in one of the accessible scopes, in which case ok is true, otherwise
// ok is false.
func (c lexicalContext) getLabel(name Name) (label Label, line int, ok bool) {
	for i := len(c) - 1; i >= 0; i-- {
		label, line, ok = c[i].getLabel(name)
		if ok {
			return
		}
	}
	return
}

// addToRoot adds name => reg mapping to the root lexical scope of this context.
func (c lexicalContext) addToRoot(name Name, reg Register) (ok bool) {
	ok = len(c) > 0
	if ok {
		c[0].reg[name] = taggedReg{reg, 0}
	}
	return
}

// addToTop adds name => reg mapping to the topmost lexical scope in this
// context.
func (c lexicalContext) addToTop(name Name, reg Register) (ok bool) {
	ok = len(c) > 0
	if ok {
		c[len(c)-1].reg[name] = taggedReg{reg, 0}
	}
	return
}

// addLabel adds a name => label mapping to the topmost lexical scope in this
// context.
func (c lexicalContext) addLabel(name Name, label Label, line int) (ok bool) {
	ok = len(c) > 0
	if ok {
		c[len(c)-1].label[name] = labelWithLine{
			Label: label,
			line:  line,
		}
	}
	return
}

// addHeight increases the height of the topmost lexical scope in this context.
func (c lexicalContext) addHeight(h int) (ok bool) {
	ok = len(c) > 0
	if ok {
		c[len(c)-1].height += h
	}
	return ok
}

func (c lexicalContext) getHeight() int {
	if len(c) > 0 {
		return c[len(c)-1].height
	}
	return 0
}

// declareGlobal adds an explicit global variable declaration to the topmost scope
func (c lexicalContext) declareGlobal(name Name, declType GlobalDeclType) (ok bool) {
	ok = len(c) > 0
	if ok {
		c[len(c)-1].globalDecls[name] = declType
	}
	return
}

// setGlobalWildcard sets the wildcard declaration type in the topmost scope
func (c lexicalContext) setGlobalWildcard(declType GlobalDeclType) (ok bool) {
	ok = len(c) > 0
	if ok {
		c[len(c)-1].globalWildcardDecl = declType
	}
	return
}

// getGlobalDeclType returns the declaration type for a global variable.
// It searches from the current scope upward, first checking for explicit declarations,
// then checking for wildcard declarations.
// - Returns MutableGlobal by default if NO global declarations exist at all (legacy Lua 5.4 behavior)
// - Returns NoDeclaredGlobal if global declarations exist but this variable is not declared
func (c lexicalContext) getGlobalDeclType(name Name) GlobalDeclType {
	hasAnyGlobalDecls := false

	// First pass: check for explicit declarations from current scope upward
	for i := len(c) - 1; i >= 0; i-- {
		globalDecls := c[i].globalDecls
		if declType, ok := globalDecls[name]; ok {
			return declType
		}
		// Track if we've seen any explicit global declarations
		if len(globalDecls) > 0 {
			hasAnyGlobalDecls = true
		}
	}

	// Second pass: check for wildcard declarations from current scope upward
	for i := len(c) - 1; i >= 0; i-- {
		if c[i].globalWildcardDecl != NoDeclaredGlobal {
			return c[i].globalWildcardDecl
		}
	}

	// If we've seen any global declarations but none matched, return NoDeclaredGlobal
	if hasAnyGlobalDecls {
		return NoDeclaredGlobal
	}

	// Default: no global declarations at all = mutable global allowed (legacy Lua 5.4 behavior)
	return MutableGlobal | LegacyGlobal
}

// pushNew returns a new LexicalContext that extends the receiver with a new
// blank lexical scope.
func (c lexicalContext) pushNew() lexicalContext {
	return append(c, lexicalScope{
		reg:                make(map[Name]taggedReg),
		label:              make(map[Name]labelWithLine),
		height:             c.top().height,
		globalDecls:        make(map[Name]GlobalDeclType),
		globalWildcardDecl: NoDeclaredGlobal,
	})
}

// pop returns a new LexicalContext that has the topmost lexical scope missing,
// and also returns that topmost scope.
func (c lexicalContext) pop() (lexicalContext, lexicalScope) {
	if len(c) == 0 {
		return c, lexicalScope{}
	}
	return c[:len(c)-1], c[len(c)-1]
}

// top returns the topmost lexical scope in the context.
func (c lexicalContext) top() lexicalScope {
	if len(c) > 0 {
		return c[len(c)-1]
	}
	return lexicalScope{}
}

func (c lexicalContext) dump() {
	for i, ns := range c {
		fmt.Printf("NS %d:\n", i)
		for name, tr := range ns.reg {
			fmt.Printf("  %s: %s\n", name, tr.reg)
		}
		// TODO: dump labels
	}
}
