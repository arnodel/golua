package code

import (
	"fmt"
	"io"
)

// A Unit is a chunk of code
type Unit struct {
	Source    string
	Code      []Opcode
	Lines     []int32
	Constants []Constant
}

func NewUnit(source string, code []Opcode, lines []int32, constants []Constant) *Unit {
	return &Unit{
		Source:    source,
		Code:      code,
		Lines:     lines,
		Constants: constants,
	}
}

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

func (d *UnitDisassembler) ShortKString(ki uint16) string {
	k := d.unit.Constants[ki]
	return k.ShortString(d)
}

func (d *UnitDisassembler) Disassemble(w io.Writer) {
	disCode := make([]string, len(d.unit.Code))
	for i, opcode := range d.unit.Code {
		disCode[i] = opcode.Disassemble(d, i)
	}
	for i, dis := range disCode {
		fmt.Fprintf(w, "%6d  %-6s  %-10s  %6d  %08x  %s\n", d.unit.Lines[i], d.labels[i], d.spans[i], i, d.unit.Code[i], dis)
	}
}
