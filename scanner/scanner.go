// Package scanner implements a tokeniser for lua.
// Inspired by https://talks.golang.org/2011/lex.slide#1
package scanner

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/arnodel/golua/token"
)

// Scanner holds the state of the scanner.
type Scanner struct {
	name             string // used only for error reports.
	input            []byte // the string being scanned.
	start, last, pos token.Pos
	items            chan *token.Token // channel of scanned items.
	state            stateFn
	errorMsg         string
}

type Option func(*Scanner)

// Specializes in scanning a number, used in file:read("n")
func ForNumber() Option {
	return func(s *Scanner) {
		s.state = scanNumberPrefix
	}
}

func WithStartLine(l int) Option {
	return func(s *Scanner) {
		pos := token.Pos{Line: l, Column: 1}
		s.start = pos
		s.pos = pos
	}
}

// New creates a new scanner for the input string.
func New(name string, input []byte, opts ...Option) *Scanner {
	l := &Scanner{
		name:  name,
		input: input,
		state: scanToken,
		items: make(chan *token.Token, 2), // Two items sufficient.
		pos:   token.Pos{Line: 1, Column: 1},
		start: token.Pos{Line: 1, Column: 1},
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// stateFn represents the state of the scanner
// as a function that returns the next state.
type stateFn func(*Scanner) stateFn

// emit passes an item back to the client.
func (l *Scanner) emit(tp token.Type) {
	lit := l.lit()
	if tp == token.INVALID {
		fmt.Println("Cannot emit", string(lit))
		panic("emit bails out")
	}
	l.items <- &token.Token{
		Type: tp,
		Lit:  lit,
		Pos:  l.start,
	}
	l.start = l.pos
}

func (l *Scanner) lit() []byte {
	return l.input[l.start.Offset:l.pos.Offset]
}

// next returns the next rune in the input.
func (l *Scanner) next() rune {
	i := l.pos.Offset
	if i >= len(l.input) {
		l.last = l.pos
		// fmt.Println("NEXT EOF")
		return -1
	}
	c, width := utf8.DecodeRune(l.input[i:])
	l.last = l.pos
	l.pos.Offset += width
	i += width
	if c == '\n' {
		if i < len(l.input) && l.input[i] == '\r' {
			l.pos.Offset++
		}
		l.pos.Line++
		l.pos.Column = 1
	} else if c == '\r' {
		if i < len(l.input) && l.input[i] == '\n' {
			l.pos.Offset++
		}
		l.pos.Line++
		l.pos.Column = 1
		c = '\n'
	} else {
		l.pos.Column++
	}
	// fmt.Println("NEXT", strconv.QuoteRune(c))
	return c
}

// ignore skips over the pending input before this point.
func (l *Scanner) ignore() {
	l.start = l.pos
	l.last = token.Pos{}
}

// backup steps back one rune.
// Can be called only once per call of next.
func (l *Scanner) backup() {
	l.pos = l.last
}

// peek returns but does not consume
// the next rune in the input.
func (l *Scanner) peek() rune {
	next := l.next()
	l.backup()
	return next
}

// accept consumes the next rune
// if it's from the valid set.
func (l *Scanner) accept(valid string) bool {
	if strings.ContainsRune(valid, l.next()) {
		return true
	}
	l.backup()
	return false
}

func (l *Scanner) acceptRune(r rune) bool {
	if l.next() == r {
		return true
	}
	l.backup()
	return false
}

// errorf returns an error token and terminates the scan
// by passing back a nil pointer that will be the next
// state, terminating l.run.
func (l *Scanner) errorf(tp token.Type, format string, args ...interface{}) stateFn {
	l.errorMsg = fmt.Sprintf(format, args...)
	l.items <- &token.Token{
		Type: tp,
		Lit:  l.lit(),
		Pos:  l.start,
	}
	return nil
}

// Scan returns the next item from the input (or nil)
func (l *Scanner) Scan() *token.Token {
	for {
		select {
		case item := <-l.items:
			return item
		default:
			if l.state == nil {
				return nil
			}
			l.state = l.state(l)
		}
	}
}

// ErrorMsg returns the current error message or an empty string if there is none.
func (l *Scanner) ErrorMsg() string {
	return l.errorMsg
}
