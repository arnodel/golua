package stringlib

import (
	"fmt"
	"math"
	"strconv"
	"unsafe"

	"github.com/arnodel/golua/lib/base"
	rt "github.com/arnodel/golua/runtime"
)

func format(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	f, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	s, err := Format(t, f, c.Etc())
	if err != nil {
		return nil, err
	}
	t.RequireBytes(len(s))
	return c.PushingNext1(t.Runtime, rt.StringValue(s)), nil
}

var errNotEnoughValues = rt.NewErrorS("not enough values for format string")

// Format is the base for the implementation of lua string.format()
//
// It works by scanning the verbs in the format string and converting the
// argument corresponding to this verb to the correct type, then calling Go's
// fmt.Sprintf().
//
// It temporarily requires all the memory needed to store the formatted string,
// but releases it before returning so the caller should require memory first
// thing after the call.
func Format(t *rt.Thread, format string, values []rt.Value) (string, *rt.Error) {
	var tmpMem uint64
	defer t.ReleaseMem(tmpMem)
	// Temporarily require memory for building the argument list
	tmpMem += t.RequireArrSize(unsafe.Sizeof(interface{}(nil)), len(values))
	args := make([]interface{}, len(values))
	j := 0
	// Temporarily require memory for building the format string
	tmpMem += t.RequireBytes(len(format))
	outFormat := []byte(format)

	// We require an amount of CPU proportional to the format string size
	t.RequireCPU(uint64(len(format)))
OuterLoop:
	for i := 0; i < len(format); i++ {
		if format[i] == '%' {
			var (
				start        = i + 1
				arg          interface{}
				length, prec int
				foundDot     bool
			)
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
					tmpMem += t.RequireBytes(1)
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
					tmpMem += t.RequireBytes(10)
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
				case 'a', 'A':
					// Hexadecimal float verbs
					outFormat[i] += 'x' - 'a'
					fallthrough
				case 'e', 'E', 'f', 'F', 'g', 'G':
					// float verbs
					if len(args) <= j {
						return "", errNotEnoughValues
					}
					f, ok := rt.ToFloat(values[j])
					if !ok {
						return "", rt.NewErrorS("invalid value for float format")
					}
					tmpMem += t.RequireBytes(10)
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
					tmpMem += t.RequireBytes(len(s))
					arg = string(s)
					break ArgLoop
				case 'q':
					// quote, only for literals I think
					if start < i {
						return "", rt.NewErrorS("specifier '%q' cannot have modifiers")
					}
					if len(args) <= j {
						return "", errNotEnoughValues
					}
					v := values[j]
					if s, ok := quote(v); ok {
						tmpMem += t.RequireBytes(len(s))
						arg = s
						outFormat[i] = 's'
					} else {
						return "", rt.NewErrorS("no literal")
					}
					break ArgLoop
				case 'p':
					// Pointer address, new in Lua 5.4
					switch v := values[j]; v.Type() {
					case rt.BoolType, rt.FloatType, rt.IntType, rt.NilType:
						outFormat[i] = 's'
						arg = "(null)"
					case rt.StringType:
						// Here we have a problem.  C Lua has one single start
						// address per string, whereas in Go a string is a
						// slice, so we can't make the same guarantee.  Best
						// effort: find the address of the start of the string
						// and format it as a pointer.
						outFormat[i] = 's'
						s := v.AsString()
						ptr := *(*uintptr)(unsafe.Pointer(&s))
						arg = fmt.Sprintf("0x%x", ptr)
					default:
						arg = v.Interface()
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
					tmpMem += t.RequireBytes(5)
					arg = bool(b)
					break ArgLoop
				case '.':
					foundDot = true
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
					if foundDot {
						prec = prec*10 + int(format[i]-'0')
						if prec >= 100 {
							return "", rt.NewErrorS("length too long")
						}
					} else {
						length = length*10 + int(format[i]-'0')
						if length >= 100 {
							return "", rt.NewErrorS("precision too long")
						}
					}
				case '+', '-', '#', ' ':
					// flag characters
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

	// Release temporary memory
	return fmt.Sprintf(string(outFormat), args...), nil
}

// Quote returns a string representing the value as a valid Lua literal if
// possible, the boolean returned indicating success or failure.
func quote(v rt.Value) (string, bool) {
	if v.IsNil() {
		return "nil", true
	}
	switch v.Type() {
	case rt.IntType:
		return strconv.Itoa(int(v.AsInt())), true
	case rt.FloatType:
		x := v.AsFloat()
		if math.IsInf(x, 0) {
			// Use an out of range literal to represent infinity.  It works
			// because that parses to infinity
			if x > 0 {
				return "1e9999", true
			} else {
				return "-1e9999", true
			}
		}
		if math.IsNaN(x) {
			return "(0/0)", true
		}
		return strconv.FormatFloat(x, 'g', -1, 64), true
	case rt.BoolType:
		return strconv.FormatBool(v.AsBool()), true
	case rt.StringType:
		return strconv.Quote(v.AsString()), true // An approximation
	default:
		return "", false
	}
}
