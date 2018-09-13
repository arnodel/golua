// Package scanner implements a tokeniser for lua.
// Inspired by https://talks.golang.org/2011/lex.slide#1
package scanner

import (
	"bytes"
	"fmt"
	"regexp"
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

// New creates a new scanner for the input string.
func New(name string, input []byte) *Scanner {
	l := &Scanner{
		name:  name,
		input: normalizeNewLines(input),
		state: scanToken,
		items: make(chan *token.Token, 2), // Two items sufficient.
		pos:   token.Pos{Line: 1, Column: 1},
		start: token.Pos{Line: 1, Column: 1},
	}
	return l
}

// stateFn represents the state of the scanner
// as a function that returns the next state.
type stateFn func(*Scanner) stateFn

// emit passes an item back to the client.
func (l *Scanner) emit(t token.Type, useMap bool) {
	lit := l.lit()
	tp := t
	if useMap {
		tp = token.TokMap.Type(string(lit))
		if tp == token.INVALID {
			tp = t
		}
	}
	if tp == token.INVALID {
		fmt.Println("Cannot emit", string(lit))
		panic("emit bails out")
	}
	// fmt.Println("EMIT", t, useMap, string(lit), tp)
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
	if l.pos.Offset >= len(l.input) {
		l.last = l.pos
		// fmt.Println("NEXT EOF")
		return -1
	}
	c, width := utf8.DecodeRune(l.input[l.pos.Offset:])
	l.last = l.pos
	l.pos.Offset += width
	if c == '\n' || c == '\r' {
		l.pos.Line++
		l.pos.Column = 1
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

// acceptRun consumes a run of runes from the valid set.
func (l *Scanner) acceptRun(valid string) {
	for strings.ContainsRune(valid, l.next()) {
	}
	l.backup()
}

// errorf returns an error token and terminates the scan
// by passing back a nil pointer that will be the next
// state, terminating l.run.
func (l *Scanner) errorf(format string, args ...interface{}) stateFn {
	l.errorMsg = fmt.Sprintf(format, args...)
	l.items <- &token.Token{
		Type: token.INVALID,
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

// Error return the current error or nil if none.
func (l *Scanner) Error() error {
	if l.errorMsg == "" {
		return nil
	}
	return fmt.Errorf("%s:%d (col %d): %s", l.name, l.pos.Line, l.pos.Column, l.errorMsg)
}

var newLines = regexp.MustCompile(`(?s)\r\n|\n\r|\r`)

func normalizeNewLines(b []byte) []byte {
	if bytes.IndexByte(b, '\r') == -1 {
		return b
	}
	return newLines.ReplaceAllLiteral(b, []byte{'\n'})
}
