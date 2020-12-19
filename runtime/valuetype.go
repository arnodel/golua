package runtime

// ValueType represents the type of a lua Value.
type ValueType uint8

// Known ValueType values.
const (
	NilType ValueType = iota
	IntType
	FloatType
	BoolType
	StringType
	CodeType
	TableType
	FunctionType
	ThreadType
	UserDataType
	UnknownType
)

// ConstantTypeMaj Constant types must be less that this.
const ConstTypeMaj = CodeType + 1
