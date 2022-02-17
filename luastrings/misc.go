package luastrings

// StringNormPos returns a normalised position in the string
// i.e. -1 -> len(s)
//      -2 -> len(s) - 1
// etc
func StringNormPos(s string, p int) int {
	if p < 0 {
		p = len(s) + 1 + p
	}
	return p
}
