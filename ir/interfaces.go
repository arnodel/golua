package ir

type Builder interface {
	Emit(Instruction, int)
	EmitJump(Name, int)
	EmitLabel(Label)
	EmitGotoLabel(Name)

	TakeRegister(Register)
	ReleaseRegister(Register)
	GetRegister(Name) (Register, bool)
	GetFreeRegister() Register

	GetNewLabel() Label

	DeclareLocal(Name, Register)
	DeclareGotoLabel(Name) Label

	PushContext()
	PopContext()

	// Doubtful
	GetConstant(Constant) uint
	GetCode() *Code
	Upvalues() []Register
	EmitNoLine(Instruction)

	// Temporary
	NewChild(string) *Compiler
}

var _ Builder = (*Compiler)(nil)
