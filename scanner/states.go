package scanner

import "github.com/arnodel/golua/token"

func scanToken(l *lexer) stateFn {
	for {
		switch c := l.next(); {
		case c == '-':
			if l.next() == '-' {
				return scanComment
			}
			l.backup()
			l.emit(-1)
		case c == '"' || c == '\'':
			return scanShortString(c)
		case isDec(c):
			l.backup()
			return scanNumber
		case c == '[':
			n := l.next()
			if n == '[' || n == '=' {
				return scanLongString
			}
			l.backup()
			l.emit(-1)
		case isAlpha(c):
			return scanIdent
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			l.ignore()
		default:
			switch c {
			case ';':
			case '(':
			case ')':
			case '=':
			case ',':
			case ':':
				l.accept(":")
			case '.':
				if l.accept(":") {
					l.accept(":")
				}
			case '<':
				l.accept("=<")
			case '>':
				l.accept("=>")
			case '|':
			case '~':
				l.accept("=")
			case '&':
			case '+':
			case '-':
			case '*':
			case '/':
				l.accept("/")
			case '%':
			case '#':
			case ']':
			case '{':
			case '}':
			default:
				return l.errorf("Unknow character")
			}
			l.emit(-1)
		}
	}
}

func scanComment(l *lexer) stateFn {
	c := l.next()
	if c == '[' {
		return scanLongComment
	}
	l.backup()
	return scanShortComment
}

func scanShortComment(l *lexer) stateFn {
	for {
		switch c := l.next(); c {
		case '\n', '\r':
			l.ignore()
			return scanToken
		case -1:
			l.ignore()
			return nil
		}
	}
}

func scanLongComment(l *lexer) stateFn {
	return scanLong(true)
}

func scanLong(comment bool) stateFn {
	return func(l *lexer) stateFn {
		level := 0
	OpeningLoop:
		for {
			switch c := l.next(); c {
			case '=':
				level += 1
			case '[':
				break OpeningLoop
			default:
				if comment {
					l.ignore()
					return scanShortComment
				}
				l.errorf("expected opening long bracket")
			}
		}
		closeLevel := -1
		// -1 means we haven't starting closing a bracket
		// 0 means we have processed the first ']'
		// n > 0 means we have processed ']' + n*'='
		for {
			switch c := l.next(); c {
			case ']':
				if closeLevel == -1 {
					closeLevel = 0
				} else if closeLevel == level {
					if comment {
						l.ignore()
					} else {
						l.emit(token.TokMap.Type("longstring"))
					}
					return scanToken
				} else {
					closeLevel = -1
				}
			case '=':
				if closeLevel >= 0 {
					closeLevel++
				}
			case -1:
				l.errorf("expected closing long bracket of level %d", level)
			default:
				closeLevel = -1
			}
		}
	}
}

func scanShortString(q rune) stateFn {
	return func(l *lexer) stateFn {
		for {
			switch c := l.next(); c {
			case q:
				l.emit(token.TokMap.Type("string"))
			case '\\':
				switch c := l.next(); {
				case c == 'x':
					if accept(l, isHex, 2) != 2 {
						return l.errorf("expected 2 hex digits")
					}
				case isDec(c):
					accept(l, isDec, 2)
				case c == 'u':
					if l.next() != '{' {
						return l.errorf("expected {")
					}
					if accept(l, isHex, -1) == 0 {
						return l.errorf("expected at least 1 hex digit")
					}
					if l.next() != '}' {
						return l.errorf("expected }")
					}
				default:
					switch c {
					case 'a', 'b', 'f', 'n', 'r', 't', 'v', 'z', '"', '\'', '\n':
						break
					default:
						// error
					}
				}
			case '\n', '\r':
				return l.errorf("unexpected newline in string literal")
			case -1:
				return l.errorf("unexpected EOF in string literal")
			}
		}
	}
}

func scanNumber(l *lexer) stateFn {
	isDigit := isDec
	exp := "eE"
	tp := token.TokMap.Type("numdec")
	if l.accept("0") && l.accept("xX") {
		isDigit = isHex
		exp = "pP"
		tp = token.TokMap.Type("numhex")
	}
	accept(l, isDigit, -1)
	if l.accept(".") {
		accept(l, isDigit, -1)
	}
	if l.accept(exp) {
		l.accept("+-")
		if accept(l, isDigit, -1) == 0 {
			return l.errorf("expected digit after exponent")
		}
	}
	if isAlpha(l.peek()) {
		return l.errorf("number followed by letter")
	}
	l.emit(tp)
	return scanToken
}

func scanLongString(l *lexer) stateFn {
	return scanLong(false)
}

func scanIdent(l *lexer) stateFn {
	accept(l, isAlnum, -1)
	l.emit(token.TokMap.Type("ident"))
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

type runePredicate func(rune) bool

func accept(l *lexer, p runePredicate, max int) int {
	for i := 0; i != max; i++ {
		if !p(l.next()) {
			l.backup()
			return i
		}
	}
	return max
}
