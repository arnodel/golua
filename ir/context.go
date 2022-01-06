package ir

import "fmt"

type lexicalScope struct {
	reg    map[Name]taggedReg
	label  map[Name]Label
	height int
}

type taggedReg struct {
	reg  Register
	tags uint
}

// A lexicalContext maintains nested mappings of names to registers and jump
// labels corresponding to nested lexical scopes.  It facilitates accessing the
// register of label associated with a given name at a given point in the code.
type lexicalContext []lexicalScope

// getRegister returns the register associated with the given name if it exists
// in one of the accessible lexical scopes.  Otherwise it sets ok to false.
// TODO: explain tags.
func (c lexicalContext) getRegister(name Name, tags uint) (reg Register, ok bool) {
	for i := len(c) - 1; i >= 0; i-- {
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
func (c lexicalContext) getLabel(name Name) (label Label, ok bool) {
	for i := len(c) - 1; i >= 0; i-- {
		label, ok = c[i].label[name]
		if ok {
			break
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
func (c lexicalContext) addLabel(name Name, label Label) (ok bool) {
	ok = len(c) > 0
	if ok {
		c[len(c)-1].label[name] = label
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

// pushNew returns a new LexicalContext that extends the receive with a new
// blank lexical scope.
func (c lexicalContext) pushNew() lexicalContext {
	return append(c, lexicalScope{
		reg:    make(map[Name]taggedReg),
		label:  make(map[Name]Label),
		height: c.top().height,
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
