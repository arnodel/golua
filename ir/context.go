package ir

import "fmt"

type lexicalMap struct {
	reg   map[Name]taggedReg
	label map[Name]Label
}

type taggedReg struct {
	reg  Register
	tags uint
}
type LexicalContext []lexicalMap

func (c LexicalContext) GetRegister(name Name, tags uint) (reg Register, ok bool) {
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

func (c LexicalContext) GetLabel(name Name) (label Label, ok bool) {
	for i := len(c) - 1; i >= 0; i-- {
		label, ok = c[i].label[name]
		if ok {
			break
		}
	}
	return
}

func (c LexicalContext) AddToRoot(name Name, reg Register) (ok bool) {
	ok = len(c) > 0
	if ok {
		c[0].reg[name] = taggedReg{reg, 0}
	}
	return
}

func (c LexicalContext) AddToTop(name Name, reg Register) (ok bool) {
	ok = len(c) > 0
	if ok {
		c[len(c)-1].reg[name] = taggedReg{reg, 0}
	}
	return
}

func (c LexicalContext) AddLabel(name Name, label Label) (ok bool) {
	ok = len(c) > 0
	if ok {
		c[len(c)-1].label[name] = label
	}
	return
}

func (c LexicalContext) PushNew() LexicalContext {
	return append(c, lexicalMap{
		reg:   make(map[Name]taggedReg),
		label: make(map[Name]Label),
	})
}

func (c LexicalContext) Pop() (LexicalContext, lexicalMap) {
	if len(c) == 0 {
		return c, lexicalMap{}
	}
	return c[:len(c)-1], c[len(c)-1]
}

func (c LexicalContext) Top() lexicalMap {
	if len(c) > 0 {
		return c[len(c)-1]
	}
	return lexicalMap{}
}

func (c LexicalContext) Dump() {
	for i, ns := range c {
		fmt.Printf("NS %d:\n", i)
		for name, tr := range ns.reg {
			fmt.Printf("  %s: %s\n", name, tr.reg)
		}
		// TODO: dump labels
	}
}
