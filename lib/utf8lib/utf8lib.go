package utf8lib

import (
	"unicode/utf8"

	rt "github.com/arnodel/golua/runtime"
)

func Load(r *rt.Runtime) {
	pkg := rt.NewTable()
	rt.SetEnv(r.GlobalEnv(), "utf8", pkg)

	rt.SetEnvGoFunc(pkg, "char", char, 0, true)
	rt.SetEnv(pkg, "charpattern", `[\0-\x7F\xC2-\xF4][\x80-\xBF]*`)

}

func char(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	runes := c.Etc()
	buf := make([]byte, len(runes)*4) // Max 4 bytes per utf8-encoded code point
	cur := buf
	bufLen := 0
	for i, r := range runes {
		n, tp := rt.ToInt(r)
		if tp != rt.IsInt {
			return nil, rt.NewErrorF("#%d should be an integer", i+1)
		}
		sz := utf8.EncodeRune(cur, rune(n))
		cur = cur[sz:]
		bufLen += sz
	}
	return c.PushingNext(rt.String(buf[:bufLen])), nil
}
