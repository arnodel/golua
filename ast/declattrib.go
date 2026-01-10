package ast

// DeclAttribType is the type of a declaration attribute
type DeclAttribType uint8

// Valid values for DeclAttribType
const (
	NoAttrib    DeclAttribType = iota
	ConstAttrib                // <const>, introduced in Lua 5.4
	CloseAttrib                // <close>, introduced in Lua 5.4
)

// DeclAttrib is a declaration attribute like <const> or <close>
type DeclAttrib struct {
	Location
	Type DeclAttribType
}

// NewDeclAttrib creates a new DeclAttrib with a location
func NewDeclAttrib(loc Location, attribType DeclAttribType) DeclAttrib {
	return DeclAttrib{
		Location: loc,
		Type:     attribType,
	}
}

// NewNoDeclAttrib creates a DeclAttrib with no attribute
func NewNoDeclAttrib() DeclAttrib {
	return DeclAttrib{
		Type: NoAttrib,
	}
}
