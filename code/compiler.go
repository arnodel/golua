package code

import "fmt"

type Label uint
type Addr uint16

type Compiler struct {
	code   []Opcode
	labels map[Label]Addr
	addrs  map[Addr]Label
}

func (c *Compiler) Emit(opcode Opcode) {
	c.code = append(c.code, opcode)
}

func (c *Compiler) EmitJump(opcode Opcode, lbl Label) {
	c.addrs[Addr(len(c.code))] = lbl
	c.Emit(opcode)
}

func (c *Compiler) EmitLabel(lbl Label) {
	c.labels[lbl] = Addr(len(c.code))
}

func (c *Compiler) ComputeJumps() error {
	for jumpFromAddr, jumpToLbl := range c.addrs {
		if jumpToAddr, ok := c.labels[jumpToLbl]; ok {
			c.code[jumpFromAddr] |= Opcode(Lit16(jumpToAddr).ToN())
		} else {
			return fmt.Errorf("Missing label: %d", jumpToLbl)
		}
	}
	return nil
}
