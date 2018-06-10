package stringlib

import (
	"fmt"

	"github.com/arnodel/golua/lib/base"
	rt "github.com/arnodel/golua/runtime"
)

func format(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	f, err := c.StringArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	s, err := Format(t, f, c.Etc())
	if err != nil {
		return nil, err.AddContext(c)
	}
	return c.PushingNext(s), nil
}

var errNotEnoughValues = rt.NewErrorS("not enough values for format string")

// Format is the base for the implementation of lua string.format()
//
// It works by scanning the verbs in the format string and converting
// the argument corresponding to this verb to the correct type, then
// calling Go's fmt.Sprintf().
func Format(t *rt.Thread, format rt.String, values []rt.Value) (rt.String, *rt.Error) {
	args := make([]interface{}, len(values))
	j := 0
OuterLoop:
	for i := 0; i < len(format); i++ {
		if format[i] == '%' {
			var arg interface{}
		ArgLoop:
			for i++; i < len(format); i++ {
				switch format[i] {
				case '%':
					continue OuterLoop
				case 'b', 'c', 'd', 'o', 'x', 'X', 'U':
					// integer verbs
					if len(args) <= j {
						return rt.String(""), errNotEnoughValues
					}
					n, tp := rt.ToInt(values[j])
					if tp != rt.IsInt {
						return rt.String(""), rt.NewErrorS("invalid value for integer format")
					}
					arg = int64(n)
					break ArgLoop
				case 'e', 'E', 'f', 'F', 'g', 'G':
					// float verbs
					if len(args) <= j {
						return rt.String(""), errNotEnoughValues
					}
					f, ok := rt.ToFloat(values[j])
					if !ok {
						return rt.String(""), rt.NewErrorS("invalid value for float format")
					}
					arg = float64(f)
					break ArgLoop
				case 's', 'q':
					// string verbs
					if len(args) <= j {
						return rt.String(""), errNotEnoughValues
					}
					s, err := base.ToString(t, values[j])
					if err != nil {
						return rt.String(""), err
					}
					arg = string(s)
					break ArgLoop
				case 't':
					// boolean verb
					if len(args) <= j {
						return rt.String(""), errNotEnoughValues
					}
					b, ok := values[j].(rt.Bool)
					if !ok {
						return rt.String(""), rt.NewErrorS("invalid value for boolean format")
					}
					arg = bool(b)
					break ArgLoop
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '+', '-', '#', ' ', '.':
					// flag characters
					continue
				default:
					// Unrecognised verbs
					return rt.String(""), rt.NewErrorS("invalid format string")
				}
			}
			args[j] = arg
			j++
		}
	}
	if j < len(args) {
		args = args[:j]
	}
	return rt.String(fmt.Sprintf(string(format), args...)), nil
}