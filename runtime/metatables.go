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

// The names of the metamethods the runtime looks up, built once.  Making the
// Value on each lookup, as StringValue(name) does, boxes the name into an
// interface every time, and a metatable is probed on every lookup that
// misses, every new key set, and every operation on a non-number.
var (
	metaIndex     = StringValue("__index")
	metaNewindex  = StringValue("__newindex")
	metaCall      = StringValue("__call")
	metaLen       = StringValue("__len")
	metaConcat    = StringValue("__concat")
	metaClose     = StringValue("__close")
	metaName      = StringValue("__name")
	metaMetatable = StringValue("__metatable")

	metaEq = StringValue("__eq")
	metaLt = StringValue("__lt")
	metaLe = StringValue("__le")

	metaAdd  = StringValue("__add")
	metaSub  = StringValue("__sub")
	metaMul  = StringValue("__mul")
	metaDiv  = StringValue("__div")
	metaIdiv = StringValue("__idiv")
	metaMod  = StringValue("__mod")
	metaPow  = StringValue("__pow")
	metaUnm  = StringValue("__unm")

	metaBand = StringValue("__band")
	metaBor  = StringValue("__bor")
	metaBxor = StringValue("__bxor")
	metaShl  = StringValue("__shl")
	metaShr  = StringValue("__shr")
	metaBnot = StringValue("__bnot")
)
