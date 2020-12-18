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
	TableType
	CodeType
	FunctionType
	ThreadType
	UserDataType
	UnknownType
)
