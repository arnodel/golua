package runtime

type Metatabler interface {
	Metatable() *Table
}

const (
	MetaFieldGcString = "__gc"
)

var (
	MetaFieldGcValue = StringValue(MetaFieldGcString)
)
