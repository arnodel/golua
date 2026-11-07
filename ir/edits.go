package ir

// Edits are changes to a function's code that a pass has decided on, kept
// apart from the code until Apply writes them out in one go: instructions
// to insert, instructions to delete, registers whose reads read another
// register instead, jumps to retarget, registers to add.  Positions are
// indices into the code the edits were made for; inserting or deleting
// doesn't shift them.
//
// A pass that collects its changes as Edits can decide in any order and
// needn't track how earlier changes move the code about; and the edits can
// be inspected before they are applied.
type Edits struct {
	inserts  map[int][]Instruction // position -> instructions to put before it
	deleted  map[int]bool          // positions of instructions that go
	rename   map[Register]Register // reads of the key read the value instead
	retarget map[int]Label         // jump at the position -> its new target
	added    []RegData             // registers added after the code's own
	nregs    int                   // registers the code has
}

// NewEdits returns empty edits for c.
func NewEdits(c Code) *Edits {
	return &Edits{
		inserts:  map[int][]Instruction{},
		deleted:  map[int]bool{},
		rename:   map[Register]Register{},
		retarget: map[int]Label{},
		nregs:    len(c.Registers),
	}
}

// InsertBefore puts instrs before position pos, after anything inserted
// there already.  Inserted instructions get no line: they stand for no
// statement.
func (e *Edits) InsertBefore(pos int, instrs ...Instruction) {
	e.inserts[pos] = append(e.inserts[pos], instrs...)
}

// Inserted returns what is to be inserted before pos so far.
func (e *Edits) Inserted(pos int) []Instruction {
	return e.inserts[pos]
}

// Delete removes the instruction at pos.
func (e *Edits) Delete(pos int) {
	e.deleted[pos] = true
}

// RenameReads makes every read of from a read of to.  Writes are not
// affected.
func (e *Edits) RenameReads(from, to Register) {
	e.rename[from] = to
}

// Retarget makes the jump at pos go to lbl.
func (e *Edits) Retarget(pos int, lbl Label) {
	e.retarget[pos] = lbl
}

// NewRegister adds a register and returns it.
func (e *Edits) NewRegister(data RegData) Register {
	e.added = append(e.added, data)
	return Register(e.nregs + len(e.added) - 1)
}

// Apply returns c with the edits made.
func (e *Edits) Apply(c Code) Code {
	out := make([]Instruction, 0, len(c.Instructions)+len(e.inserts))
	lines := make([]int, 0, len(c.Instructions)+len(e.inserts))
	for i := 0; i <= len(c.Instructions); i++ {
		for _, in := range e.inserts[i] {
			out = append(out, in)
			lines = append(lines, 0)
		}
		if i == len(c.Instructions) || e.deleted[i] {
			continue
		}
		instr := c.Instructions[i]
		for _, r := range instr.Reads() {
			if to, ok := e.rename[r]; ok {
				instr = instr.ReplaceRead(r, to)
			}
		}
		if to, ok := e.retarget[i]; ok {
			switch x := instr.(type) {
			case Jump:
				x.Label = to
				instr = x
			case JumpIf:
				x.Label = to
				instr = x
			}
		}
		out = append(out, instr)
		lines = append(lines, c.Lines[i])
	}
	c.Instructions = out
	c.Lines = lines
	if len(e.added) > 0 {
		c.Registers = append(c.Registers[:len(c.Registers):len(c.Registers)], e.added...)
	}
	return c
}
