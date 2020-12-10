package code

import (
	"fmt"
	"io"
)

// A Unit is a chunk of code with associated constants.
type Unit struct {
	Source    string     // Shows were the unit comes from (e.g. a filename) - only for information.
	Code      []Opcode   // The code
	Lines     []int32    // Optional: source code line for the corresponding opcode
	Constants []Constant // All the constants required for running the code
}

// Disassemble outputs the disassembly of the unit code into the given
// io.Writer.
func (u *Unit) Disassemble(w io.Writer) {
	newUnitDisassembler(u).disassemble(w)
}

type unitDisassembler struct {
	unit   *Unit
	labels map[int]string
	spans  map[int]string
}

var _ OpcodeDisassembler = (*unitDisassembler)(nil)

func newUnitDisassembler(unit *Unit) *unitDisassembler {
	return &unitDisassembler{
		unit:   unit,
		labels: make(map[int]string),
		spans:  make(map[int]string),
	}
}

func (d *unitDisassembler) disassemble(w io.Writer) {
	fmt.Fprintf(w, "==CONSTANTS==\n\n")
	for i, k := range d.unit.Constants {
		fmt.Fprintf(w, "K%d = %s\n", i, k.ShortString())
		if sg, ok := k.(spanGetter); ok {
			d.setSpan(sg.GetSpan())
		}
	}
	disCode := make([]string, len(d.unit.Code))
	maxSpanLen := 10
	for i, opcode := range d.unit.Code {
		disCode[i] = opcode.Disassemble(d, i)
		if l := len(d.spans[i]); l > maxSpanLen {
			maxSpanLen = l
		}
	}
	fmt.Fprintf(w, "\n==CODE==\n\n")
	for i, dis := range disCode {
		fmt.Fprintf(w, "%6d  %-6s  %-*s  %6d  %08x  %s\n", d.unit.Lines[i], d.labels[i], maxSpanLen, d.spans[i], i, d.unit.Code[i], dis)
	}
}

func (d *unitDisassembler) GetLabel(offset int) string {
	lbl, ok := d.labels[offset]
	if !ok {
		lbl = fmt.Sprintf("L%d", len(d.labels))
		d.labels[offset] = lbl
	}
	return lbl
}

func (d *unitDisassembler) setSpan(name string, startOffset, endOffset int) {
	d.spans[startOffset] = name
	for {
		startOffset++
		d.spans[startOffset] = " |"
		if startOffset == endOffset {
			d.spans[endOffset] = `  \`
			return
		}
	}
}

func (d *unitDisassembler) ShortKString(ki KIndex) string {
	k := d.unit.Constants[ki]
	return k.ShortString()
}

// Interface for constants that contain code to notify the disassemble of where
// this code is (and how to label it).
type spanGetter interface {
	GetSpan() (string, int, int)
}
