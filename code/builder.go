package code

// A Label represent a location in the code.
type Label uint

// A Builder helps build a code Unit (in particular it calculates the offsets
// for jump instructions).
type Builder struct {
	source    string          // identifies the source of the code
	lines     []int32         // lines in the source code corresponding to the opcodes
	code      []Opcode        // opcodes emitted
	jumpTo    map[Label]int   // destination locations for the labels
	jumpFrom  map[Label][]int // lists of locations for opcode that jump to a given label
	constants []Constant      // constants required for the code
}

// NewBuilder returns an empty Builder for the given source.
func NewBuilder(source string) *Builder {
	return &Builder{
		source:   source,
		jumpTo:   make(map[Label]int),
		jumpFrom: make(map[Label][]int),
	}
}

// Emit adds an opcode (associating it with a source code line).
func (c *Builder) Emit(opcode Opcode, line int) {
	c.code = append(c.code, opcode)
	c.lines = append(c.lines, int32(line))
}

// EmitJump adds a jump opcode, jumping to the given label.  The offset part of
// the opcode must be left as 0, it will be filled by the builder when the
// location of the label is known.
func (c *Builder) EmitJump(opcode Opcode, lbl Label, line int) {
	jumpToAddr, ok := c.jumpTo[lbl]
	addr := len(c.code)
	if ok {
		opcode |= Opcode(Lit16(jumpToAddr - addr).ToN())
	} else {
		c.jumpFrom[lbl] = append(c.jumpFrom[lbl], addr)
	}
	c.Emit(opcode, line)
}

// EmitLabel adds a label for the current location.  It panics if called twice
// with the same label at different locations.
func (c *Builder) EmitLabel(lbl Label) {
	addr := len(c.code)
	if lblAddr, ok := c.jumpTo[lbl]; ok && lblAddr != addr {
		panic("Label already emitted for a different location")
	}
	c.jumpTo[lbl] = addr
	for _, jumpFromAddr := range c.jumpFrom[lbl] {
		c.code[jumpFromAddr] |= Opcode(Lit16(addr - jumpFromAddr).ToN())
	}
	delete(c.jumpFrom, lbl)
}

// Offset returns the current location.  It must be called when all emitted jump
// labels have been resolved, otherwise it panice.
func (c *Builder) Offset() uint {
	if len(c.jumpFrom) > 0 {
		panic("Illegal offset")
	}
	c.jumpTo = make(map[Label]int)
	return uint(len(c.code))
}

// AddConstant adds a constant.
func (c *Builder) AddConstant(k Constant) {
	c.constants = append(c.constants, k)
}

// GetUnit returns the build code Unit.
func (c *Builder) GetUnit() *Unit {
	return &Unit{
		Source:    c.source,
		Code:      c.code,
		Lines:     c.lines,
		Constants: c.constants,
	}
}
