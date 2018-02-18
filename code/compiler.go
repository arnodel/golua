package code

type Label uint

type Compiler struct {
}

func (c *Compiler) Emit(opcode Opcode) {
}

func (c *Compiler) EmitRecv(withEtc bool, dst ...[]Reg) {

	for i := 0; i < len(dst); i++ {

	}
}

func (c *Compiler) GetLabelAddr(lbl Label) uint16 {
	return 0
}

func (c *Compiler) SetLableAddr(lbl Label) {
}
