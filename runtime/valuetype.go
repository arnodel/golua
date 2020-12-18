package runtime

type ValueType uint8

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
