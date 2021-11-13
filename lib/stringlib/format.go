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
	t.RequireMem(uint64(len(s)))
	return c.PushingNext(rt.StringValue(s)), nil
}

var errNotEnoughValues = rt.NewErrorS("not enough values for format string")

// Format is the base for the implementation of lua string.format()
//
// It works by scanning the verbs in the format string and converting
// the argument corresponding to this verb to the correct type, then
// calling Go's fmt.Sprintf().
func Format(t *rt.Thread, format string, values []rt.Value) (string, *rt.Error) {
	args := make([]interface{}, len(values))
	j := 0
	outFormat := []byte(format)
OuterLoop:
	for i := 0; i < len(format); i++ {
		if format[i] == '%' {
			var arg interface{}
		ArgLoop:
			for i++; i < len(format); i++ {
				switch format[i] {
				case '%':
					continue OuterLoop
				case 'c':
					if len(args) <= j {
						return "", errNotEnoughValues
					}
					n, ok := rt.ToInt(values[j])
					if !ok {
						return "", rt.NewErrorS("invalid value for integer format")
					}
					arg = []byte{byte(n)}
					outFormat[i] = 's'
					break ArgLoop
				case 'b', 'd', 'o', 'x', 'X', 'U', 'i', 'u':
					// integer verbs
					if len(args) <= j {
						return "", errNotEnoughValues
					}
					n, ok := rt.ToInt(values[j])
					if !ok {
						return "", rt.NewErrorS("invalid value for integer format")
					}
					switch format[i] {
					case 'u':
						// Unsigned int
						arg = uint64(n)
						outFormat[i] = 'd' // No 'u' verb in Go
					case 'i':
						// Signed int
						arg = int64(n)
						outFormat[i] = 'd' // No 'i' verb in Go
					case 'x', 'X':
						arg = uint64(n) // Need to convert to unsigned
					default:
						arg = int64(n)
					}
					break ArgLoop
				case 'e', 'E', 'f', 'F', 'g', 'G':
					// float verbs
					if len(args) <= j {
						return "", errNotEnoughValues
					}
					f, ok := rt.ToFloat(values[j])
					if !ok {
						return "", rt.NewErrorS("invalid value for float format")
					}
					arg = float64(f)
					break ArgLoop
				case 's':
					if len(args) <= j {
						return "", errNotEnoughValues
					}
					s, err := base.ToString(t, values[j])
					if err != nil {
						return "", err
					}
					arg = string(s)
					break ArgLoop
				case 'q':
					// quote, only for literals I think
					if len(args) <= j {
						return "", errNotEnoughValues
					}
					v := values[j]
					if s, ok := v.TryString(); ok {
						arg = string(s)
					} else {
						s, ok := rt.ToString(v)
						if !ok && s == "" {
							return "", rt.NewErrorS("no literal")
						}
						arg = string(s)
						// Not a string, print verbatim
						outFormat[i] = 's'
					}
					break ArgLoop
				case 't':
					// boolean verb
					if len(args) <= j {
						return "", errNotEnoughValues
					}
					b, ok := values[j].TryBool()
					if !ok {
						return "", rt.NewErrorS("invalid value for boolean format")
					}
					arg = bool(b)
					break ArgLoop
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '+', '-', '#', ' ', '.':
					// flag characters
					continue
				default:
					// Unrecognised verbs
					return "", rt.NewErrorS("invalid format string")
				}
			}
			args[j] = arg
			j++
		}
	}
	if j < len(args) {
		args = args[:j]
	}
	return fmt.Sprintf(string(outFormat), args...), nil
}
