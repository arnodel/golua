package stringlib

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/arnodel/golua/lib/stringlib/pattern"
	rt "github.com/arnodel/golua/runtime"
)

func find(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var (
		s, ptn string
		init   int64 = 1
		plain  bool
	)
	err := c.CheckNArgs(2)
	if err == nil {
		s, err = c.StringArg(0)
	}
	if err == nil {
		ptn, err = c.StringArg(1)
	}
	if err == nil && c.NArgs() >= 3 {
		init, err = c.IntArg(2)
		if err == nil && c.NArgs() >= 4 {
			plain = rt.Truth(c.Arg(3))
		}
	}
	if err != nil {
		return nil, err
	}
	si := rt.StringNormPos(s, int(init)) - 1
	next := c.Next()
	switch {
	case si < 0 || si > len(s):
		t.Push1(next, rt.NilValue)
	case plain || len(ptn) == 0:
		// strings.Index is linear
		t.RequireCPU(uint64(len(s) - si))
		i := strings.Index(s[si:], ptn)
		if i == -1 {
			t.Push1(next, rt.NilValue)
		} else {
			t.Push1(next, rt.IntValue(int64(i+1)))
			t.Push1(next, rt.IntValue(int64(i+len(ptn))))
		}
	default:
		pat, err := pattern.New(string(ptn))
		if err != nil {
			return nil, rt.NewErrorE(err)
		}
		captures, usedCPU := pat.MatchFromStart(string(s), si, t.UnusedCPU())
		t.RequireCPU(usedCPU)
		if len(captures) == 0 {
			t.Push1(next, rt.NilValue)
		} else {
			first := captures[0]
			t.Push1(next, rt.IntValue(int64(first.Start()+1)))
			t.Push1(next, rt.IntValue(int64(first.End())))
			pushExtraCaptures(t.Runtime, captures, s, next)
		}
	}
	return next, nil
}

func match(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var (
		s, ptn string
		init   int64 = 1
	)
	err := c.CheckNArgs(2)
	if err == nil {
		s, err = c.StringArg(0)
	}
	if err == nil {
		ptn, err = c.StringArg(1)
	}
	if err == nil && c.NArgs() >= 3 {
		init, err = c.IntArg(2)
	}
	if err != nil {
		return nil, err
	}
	si := rt.StringNormPos(s, int(init)) - 1
	next := c.Next()
	pat, ptnErr := pattern.New(string(ptn))
	if ptnErr != nil {
		return nil, rt.NewErrorE(ptnErr)
	}
	captures, usedCPU := pat.MatchFromStart(string(s), si, t.UnusedCPU())
	t.RequireCPU(usedCPU)
	pushCaptures(t.Runtime, captures, s, next)
	return next, nil
}

func pushCaptures(r *rt.Runtime, captures []pattern.Capture, s string, next rt.Cont) {
	switch len(captures) {
	case 0:
		r.Push1(next, rt.NilValue)
	case 1:
		c := captures[0]
		r.RequireBytes(c.End() - c.Start())
		r.Push1(next, rt.StringValue(s[c.Start():c.End()]))
	default:
		pushExtraCaptures(r, captures, s, next)
	}
}

func pushExtraCaptures(r *rt.Runtime, captures []pattern.Capture, s string, next rt.Cont) {
	if len(captures) < 2 {
		return
	}
	for _, c := range captures[1:] {
		r.Push1(next, captureValue(r, c, s))
	}
}

func captureValue(r *rt.Runtime, c pattern.Capture, s string) rt.Value {
	if c.IsEmpty() {
		return rt.IntValue(int64(c.Start() + 1))
	}
	r.RequireBytes(c.End() - c.Start())
	return rt.StringValue(s[c.Start():c.End()])
}

func gmatch(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var s, ptn string
	err := c.CheckNArgs(2)
	if err == nil {
		s, err = c.StringArg(0)
	}
	if err == nil {
		ptn, err = c.StringArg(1)
	}
	if err != nil {
		return nil, err
	}
	pat, ptnErr := pattern.New(string(ptn))
	if ptnErr != nil {
		return nil, rt.NewErrorE(ptnErr)
	}
	si := 0
	allowEmpty := true
	var iterator = func(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
		next := c.Next()
		var (
			captures []pattern.Capture
			usedCPU  uint64
		)
		for {
			captures, usedCPU = pat.Match(string(s), si, t.UnusedCPU())
			t.RequireCPU(usedCPU)
			if len(captures) == 0 {
				break
			}
			gc := captures[0]
			start, end := gc.Start(), gc.End()
			if allowEmpty || start != si || end != si {
				allowEmpty = start >= end
				if allowEmpty {
					si = start + 1
				} else {
					si = end
				}
				break
			}
			si++
			allowEmpty = true
		}
		pushCaptures(t.Runtime, captures, s, next)
		return next, nil
	}
	iterGof := rt.NewGoFunction(iterator, "gmatchiterator", 0, false)
	iterGof.SolemnlyDeclareCompliance(rt.ComplyCpuSafe | rt.ComplyMemSafe | rt.ComplyTimeSafe | rt.ComplyIoSafe)
	return c.PushingNext(t.Runtime, rt.FunctionValue(iterGof)), nil
}

func gsub(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	var (
		s, ptn string
		n      int64 = -1
		repl   rt.Value
	)
	err := c.CheckNArgs(3)
	if err == nil {
		s, err = c.StringArg(0)
	}
	if err == nil {
		ptn, err = c.StringArg(1)
	}
	if err == nil && c.NArgs() >= 4 {
		n, err = c.IntArg(3)
	}
	if err != nil {
		return nil, err
	}
	repl = c.Arg(2)
	pat, ptnErr := pattern.New(string(ptn))
	if ptnErr != nil {
		return nil, rt.NewErrorE(ptnErr)
	}

	// replF will be the function that does the substitution of the match given
	// the captures in the match.  It must require the memory for the
	// substitution string
	var replF func([]pattern.Capture) (string, *rt.Error)

	if replString, ok := repl.TryString(); ok {
		replF = func(captures []pattern.Capture) (string, *rt.Error) {
			cStrings := [10]string{}
			maxIndex := len(captures) - 1
			for i, c := range captures {
				v := captureValue(t.Runtime, c, s)
				switch v.Type() {
				case rt.StringType:
					cStrings[i] = v.AsString()
				case rt.IntType:
					cStrings[i] = strconv.Itoa(int(v.AsInt()))
				}
			}
			if len(captures) == 1 {
				cStrings[1] = cStrings[0]
				maxIndex = 1
			}
			var err *rt.Error
			t.RequireCPU(uint64(len(replString)))
			t.RequireBytes(len(replString))
			return gsubPtn.ReplaceAllStringFunc(replString, func(x string) string {
				if err != nil {
					return ""
				}
				b := x[1]
				switch {
				case '0' <= b && b <= '9':
					idx := int(b - '0')
					if idx > maxIndex {
						err = rt.NewErrorE(pattern.ErrInvalidCaptureIdx(idx))
						return ""
					}
					s := cStrings[b-'0']
					if len(s) > 2 {
						t.RequireBytes(len(s) - 2)
					}
					return s
				case b == '%':
					return x[1:]
				default:
					err = rt.NewErrorE(pattern.ErrInvalidPct)
				}
				return x[1:]
			}), err
		}
	} else if replTable, ok := repl.TryTable(); ok {
		replF = func(captures []pattern.Capture) (string, *rt.Error) {
			gc := captures[0]
			i := 0
			if len(captures) >= 2 {
				i = 1
			}
			c := captures[i]
			val, err := rt.Index(t, rt.TableValue(replTable), captureValue(t.Runtime, c, s))
			if err != nil {
				return "", err
			}
			return subToString(t.Runtime, s[gc.Start():gc.End()], val)
		}
	} else if replC, ok := repl.TryCallable(); ok {
		replF = func(captures []pattern.Capture) (string, *rt.Error) {
			term := rt.NewTerminationWith(c, 1, false)
			cont := replC.Continuation(t.Runtime, term)
			gc := captures[0]
			i := 0
			if len(captures) >= 2 {
				i = 1
			}
			for _, c := range captures[i:] {
				t.Push1(cont, captureValue(t.Runtime, c, s))
			}
			err := t.RunContinuation(cont)
			if err != nil {
				return "", err
			}
			return subToString(t.Runtime, s[gc.Start():gc.End()], term.Get(0))
		}
	} else {
		return nil, rt.NewErrorS("#3 must be a string, table or function")
	}
	var (
		si         int
		sb         strings.Builder
		matchCount int64
		allowEmpty = true
	)
	// We require memory for the string we build as we go along
	for ; matchCount != n; matchCount++ {
		captures, usedCPU := pat.Match(string(s), si, t.UnusedCPU())
		t.RequireCPU(usedCPU)
		if len(captures) == 0 {
			break
		}
		gc := captures[0]
		start, end := gc.Start(), gc.End()
		if allowEmpty || start != si || end != si {
			sub, err := replF(captures)
			if err != nil {
				return nil, err
			}
			t.RequireBytes(si - start)
			// No need to require memory for sub as that has been done already
			// by replF
			_, _ = sb.WriteString(s[si:start])
			_, _ = sb.WriteString(sub)
		}
		allowEmpty = start >= end
		if allowEmpty {
			if start < len(s) {
				t.RequireBytes(1)
				_ = sb.WriteByte(s[start])
			}
			si = start + 1
		} else {
			si = end
		}
	}
	if si < len(s) {
		t.RequireBytes(len(s) - si)
		_, _ = sb.WriteString(s[si:])
	}
	next := c.Next()
	// Already required memory for the string below.
	t.Push1(next, rt.StringValue(sb.String()))
	t.Push1(next, rt.IntValue(matchCount))
	return next, nil
}

var gsubPtn = regexp.MustCompile("%.")

func subToString(r *rt.Runtime, key string, val rt.Value) (string, *rt.Error) {
	if !rt.Truth(val) {
		return key, nil
	}
	res, ok := val.ToString()
	if ok {
		r.RequireBytes(len(res))
		return res, nil
	}
	return "", rt.NewErrorF("invalid replacement value (a %s)", val.TypeName())
}
