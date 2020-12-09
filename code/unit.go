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
	NewUnitDisassembler(u).Disassemble(w)
}

type UnitDisassembler struct {
	unit   *Unit
	labels map[int]string
	spans  map[int]string
}

func NewUnitDisassembler(unit *Unit) *UnitDisassembler {
	return &UnitDisassembler{
		unit:   unit,
		labels: make(map[int]string),
		spans:  make(map[int]string),
	}
}

func (d *UnitDisassembler) SetLabel(offset int, lbl string) {
	d.labels[offset] = lbl
}

func (d *UnitDisassembler) GetLabel(offset int) string {
	lbl, ok := d.labels[offset]
	if !ok {
		lbl = fmt.Sprintf("L%d", len(d.labels))
		d.labels[offset] = lbl
	}
	return lbl
}

func (d *UnitDisassembler) SetSpan(name string, startOffset, endOffset int) {
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

func (d *UnitDisassembler) ShortKString(ki KIndex) string {
	k := d.unit.Constants[ki]
	return k.ShortString()
}

type spanGetter interface {
	GetSpan() (string, int, int)
}

func (d *UnitDisassembler) Disassemble(w io.Writer) {
	for _, k := range d.unit.Constants {
		if sg, ok := k.(spanGetter); ok {
			d.SetSpan(sg.GetSpan())
		}
	}
	disCode := make([]string, len(d.unit.Code))
	for i, opcode := range d.unit.Code {
		disCode[i] = opcode.Disassemble(d, i)
	}
	for i, dis := range disCode {
		fmt.Fprintf(w, "%6d  %-6s  %-10s  %6d  %08x  %s\n", d.unit.Lines[i], d.labels[i], d.spans[i], i, d.unit.Code[i], dis)
	}
}
