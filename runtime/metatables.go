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

// A metaID names one of the metamethods the runtime looks up.  Looking a
// metamethod up by id rather than by name lets a table that is a metatable
// keep the answers (see metaCache).
type metaID uint8

const (
	metaIndex metaID = iota
	metaNewindex
	metaCall
	metaLen
	metaConcat
	metaClose
	metaName
	metaMetatable

	metaEq
	metaLt
	metaLe

	metaAdd
	metaSub
	metaMul
	metaDiv
	metaIdiv
	metaMod
	metaPow
	metaUnm

	metaBand
	metaBor
	metaBxor
	metaShl
	metaShr
	metaBnot
	metaGc

	numMeta
)

// metaNames holds the name of each metamethod as a Value, built once: making
// the Value on each lookup, as StringValue(name) does, boxes the name into an
// interface every time.
var metaNames = func() (names [numMeta]Value) {
	for id, name := range map[metaID]string{
		metaIndex: "__index", metaNewindex: "__newindex", metaCall: "__call",
		metaLen: "__len", metaConcat: "__concat", metaClose: "__close",
		metaName: "__name", metaMetatable: "__metatable",
		metaEq: "__eq", metaLt: "__lt", metaLe: "__le",
		metaAdd: "__add", metaSub: "__sub", metaMul: "__mul", metaDiv: "__div",
		metaIdiv: "__idiv", metaMod: "__mod", metaPow: "__pow", metaUnm: "__unm",
		metaBand: "__band", metaBor: "__bor", metaBxor: "__bxor",
		metaShl: "__shl", metaShr: "__shr", metaBnot: "__bnot",
		metaGc: MetaFieldGcString,
	} {
		names[id] = StringValue(name)
	}
	return names
}()

// metaIDs maps a metamethod's name to its id.
var metaIDs = func() map[string]metaID {
	ids := map[string]metaID{}
	for id, name := range metaNames {
		ids[name.AsString()] = metaID(id)
	}
	return ids
}()

// A metaCache holds the metamethods of a table that is used as a metatable,
// as the runtime finds them: a value, or nil for an absent one, which is the
// usual answer (every store of a new key into a table with a metatable asks
// for __newindex).  Looking a metamethod up is then an array read in place
// of a string lookup, and the cache is cleared by any write to the table.
//
// Only tables that have been set as the metatable of something carry one;
// the others pay a nil check on each write, nothing else.
type metaCache struct {
	known uint32 // which of vals are filled
	vals  [numMeta]Value
}

// fill looks metamethod id up in t, the table c belongs to, and keeps the
// answer.
func (c *metaCache) fill(t *Table, id metaID) Value {
	v := t.getString(metaNames[id])
	c.vals[id] = v
	c.known |= 1 << id
	return v
}
