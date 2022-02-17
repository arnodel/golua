package scanner

import (
	"github.com/arnodel/golua/token"
)

func scanToken(l *Scanner) stateFn {
	for {
		switch c := l.next(); {
		case c == '-':
			if l.next() == '-' {
				return scanComment
			}
			l.backup()
			l.emit(token.SgMinus)
		case c == '"' || c == '\'':
			return scanShortString(c)
		case isDec(c):
			l.backup()
			return scanNumber
		case c == '[':
			n := l.next()
			if n == '[' || n == '=' {
				l.backup()
				return scanLongString
			}
			l.backup()
			l.emit(token.SgOpenSquareBkt)
		case isAlpha(c):
			return scanIdent
		case isSpace(c):
			l.ignore()
		default:
			switch c {
			case ';', '(', ')', ',', '|', '&', '+', '*', '%', '^', '#', ']', '{', '}':
			case '=':
				l.accept("=")
			case ':':
				l.accept(":")
			case '.':
				if accept(l, isDec, -1) > 0 {
					return scanExp(l, isDec, "eE", token.NUMDEC)
				}
				if l.accept(".") {
					l.accept(".")
				}
			case '<':
				l.accept("=<")
			case '>':
				l.accept("=>")
			case '~':
				l.accept("=")
			case '/':
				l.accept("/")
			case -1:
				l.emit(token.EOF)
				return nil
			default:
				return l.errorf(token.INVALID, "illegal character")
			}
			l.emit(sgType[string(l.lit())])
		}
		return scanToken
	}
}

func scanComment(l *Scanner) stateFn {
	c := l.next()
	if c == '[' {
		return scanLongComment
	}
	l.backup()
	return scanShortComment
}

func scanShortComment(l *Scanner) stateFn {
	for {
		switch c := l.next(); c {
		case '\n':
			l.acceptRune('\r')
			l.ignore()
			return scanToken
		case -1:
			l.ignore()
			l.emit(token.EOF)
			return nil
		}
	}
}

func scanLongComment(l *Scanner) stateFn {
	return scanLong(true)
}

func scanLong(comment bool) stateFn {
	return func(l *Scanner) stateFn {
		level := 0
	OpeningLoop:
		for {
			switch c := l.next(); c {
			case '=':
				level++
			case '[':
				break OpeningLoop
			default:
				if comment {
					l.ignore()
					return scanShortComment
				}
				return l.errorf(token.INVALID, "expected opening long bracket")
			}
		}
		closeLevel := -1
		// -1 means we haven't starting closing a bracket
		// 0 means we have processed the first ']'
		// n > 0 means we have processed ']' + n*'='
		for {
			switch c := l.next(); c {
			case ']':
				if closeLevel == level {
					if comment {
						l.ignore()
					} else {
						l.emit(token.LONGSTRING)
					}
					return scanToken
				}
				closeLevel = 0
			case '=':
				if closeLevel >= 0 {
					closeLevel++
				}
			case -1:
				return l.errorf(token.UNFINISHED, "illegal <eof> in long bracket of level %d", level)
			default:
				closeLevel = -1
			}
		}
	}
}

func scanShortString(q rune) stateFn {
	return func(l *Scanner) stateFn {
		for {
			switch c := l.next(); c {
			case q:
				l.emit(token.STRING)
				return scanToken
			case '\\':
				switch c := l.next(); {
				case c == 'x':
					if accept(l, isHex, 2) != 2 {
						return l.errorf(token.INVALID, `\x must be followed by 2 hex digits`)
					}
				case isDec(c):
					accept(l, isDec, 2)
				case c == 'u':
					if l.next() != '{' {
						return l.errorf(token.INVALID, `\u must be followed by '{'`)
					}
					if accept(l, isHex, -1) == 0 {
						return l.errorf(token.INVALID, "at least 1 hex digit required")
					}
					if l.next() != '}' {
						return l.errorf(token.INVALID, "missing '}'")
					}
				case c == 'z':
					accept(l, isSpace, -1)
				default:
					switch c {
					case '\n':
						// Nothing to do
					case 'a', 'b', 'f', 'n', 'r', 't', 'v', 'z', '"', '\'', '\\':
						break
					default:
						return l.errorf(token.INVALID, "illegal escaped character")
					}
				}
			case '\n', '\r':
				return l.errorf(token.INVALID, "illegal new line in string literal")
			case -1:
				return l.errorf(token.INVALID, "illegal <eof> in string literal")
			}
		}
	}
}

// For scanning numbers e.g. in files
func scanNumberPrefix(l *Scanner) stateFn {
	accept(l, isSpace, -1)
	l.accept("+-")
	return scanNumber
}

func scanNumber(l *Scanner) stateFn {
	isDigit := isDec
	exp := "eE"
	tp := token.NUMDEC
	leading0 := l.accept("0")
	dcount := 0
	if leading0 && l.accept("xX") {
		isDigit = isHex
		exp = "pP"
		tp = token.NUMHEX
	} else if leading0 {
		dcount++
	}
	dcount += accept(l, isDigit, -1)
	if l.accept(".") {
		dcount += accept(l, isDigit, -1)
	}
	if dcount == 0 {
		return l.errorf(token.INVALID, "no digits in mantissa")
	}
	return scanExp(l, isDigit, exp, tp)
}

func scanExp(l *Scanner, isDigit func(rune) bool, exp string, tp token.Type) stateFn {
	if l.accept(exp) {
		l.accept("+-")
		if accept(l, isDec, -1) == 0 {
			return l.errorf(token.INVALID, "digit required after exponent")
		}
	}
	l.emit(tp)
	if isAlpha(l.peek()) {
		l.next()
		return l.errorf(token.INVALID, "illegal character following number")
	}
	return scanToken
}

func scanLongString(l *Scanner) stateFn {
	return scanLong(false)
}

var kwType = map[string]token.Type{
	"break":    token.KwBreak,
	"goto":     token.KwGoto,
	"do":       token.KwDo,
	"while":    token.KwWhile,
	"end":      token.KwEnd,
	"repeat":   token.KwRepeat,
	"until":    token.KwUntil,
	"then":     token.KwThen,
	"else":     token.KwElse,
	"elseif":   token.KwElseIf,
	"if":       token.KwIf,
	"for":      token.KwFor,
	"in":       token.KwIn,
	"function": token.KwFunction,
	"local":    token.KwLocal,
	"and":      token.KwAnd,
	"or":       token.KwOr,
	"not":      token.KwNot,
	"nil":      token.KwNil,
	"true":     token.KwTrue,
	"false":    token.KwFalse,
	"return":   token.KwReturn,
}

var sgType = map[string]token.Type{
	"-":  token.SgMinus,
	"+":  token.SgPlus,
	"*":  token.SgStar,
	"/":  token.SgSlash,
	"//": token.SgSlashSlash,
	"%":  token.SgPct,
	"|":  token.SgPipe,
	"&":  token.SgAmpersand,
	"^":  token.SgHat,
	">>": token.SgShiftRight,
	"<<": token.SgShiftLeft,
	"..": token.SgConcat,

	"==": token.SgEqual,
	"~=": token.SgNotEqual,
	"<":  token.SgLess,
	"<=": token.SgLessEqual,
	">":  token.SgGreater,
	">=": token.SgGreaterEqual,

	"...": token.SgEtc,

	"[":  token.SgOpenSquareBkt,
	"]":  token.SgCloseSquareBkt,
	"(":  token.SgOpenBkt,
	")":  token.SgCloseBkt,
	"{":  token.SgOpenBrace,
	"}":  token.SgCloseBrace,
	";":  token.SgSemicolon,
	",":  token.SgComma,
	".":  token.SgDot,
	":":  token.SgColon,
	"::": token.SgDoubleColon,
	"=":  token.SgAssign,
	"#":  token.SgHash,
	"~":  token.SgTilde,
}

func scanIdent(l *Scanner) stateFn {
	accept(l, isAlnum, -1)
	tp, ok := kwType[string(l.lit())]
	if !ok {
		tp = token.IDENT
	}
	l.emit(tp)
	return scanToken
}

func isDec(x rune) bool {
	return '0' <= x && x <= '9'
}

func isAlpha(x rune) bool {
	return x >= 'a' && x <= 'z' || x >= 'A' && x <= 'Z' || x == '_'
}

func isAlnum(x rune) bool {
	return isDec(x) || isAlpha(x)
}

func isHex(x rune) bool {
	return isDec(x) || 'a' <= x && x <= 'f' || 'A' <= x && x <= 'F'
}

func isSpace(x rune) bool {
	return x == ' ' || x == '\n' || x == '\r' || x == '\t' || x == '\v' || x == '\f'
}

type runePredicate func(rune) bool

func accept(l *Scanner, p runePredicate, max int) int {
	for i := 0; i != max; i++ {
		if !p(l.next()) {
			l.backup()
			return i
		}
	}
	return max
}
