package scanner

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/arnodel/golua/token"
)

// lexer holds the state of the scanner.
type lexer struct {
	name             string // used only for error reports.
	input            []byte // the string being scanned.
	start, last, pos token.Pos
	items            chan *token.Token // channel of scanned items.
	state            stateFn
}

// stateFn represents the state of the scanner
// as a function that returns the next state.
type stateFn func(*lexer) stateFn

// emit passes an item back to the client.
func (l *lexer) emit(t token.Type) {
	lit := l.input[l.start.Offset:l.pos.Offset]
	if t == -1 {
		t = token.TokMap.Type(string(lit))
	}
	l.items <- &token.Token{
		Type: t,
		Lit:  lit,
		Pos:  l.start,
	}
	l.start = l.pos
}

// next returns the next rune in the input.
func (l *lexer) next() rune {
	if l.pos.Offset >= len(l.input) {
		l.last = token.Pos{}
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
	return c
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
	l.last = token.Pos{}
}

// backup steps back one rune.
// Can be called only once per call of next.
func (l *lexer) backup() {
	if l.last == (token.Pos{}) {
		panic("oops")
	}
	l.pos = l.last
}

// peek returns but does not consume
// the next rune in the input.
func (l *lexer) peek() rune {
	rune := l.next()
	l.backup()
	return rune
}

// accept consumes the next rune
// if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

// acceptRun consumes a run of runes from the valid set.
func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

// error returns an error token and terminates the scan
// by passing back a nil pointer that will be the next
// state, terminating l.run.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- &token.Token{
		Type: token.INVALID,
		Lit:  []byte(fmt.Sprintf(format, args...)),
		Pos:  l.pos,
	}
	return nil
}

// lex creates a new scanner for the input string.
func lex(name string, input []byte) *lexer {
	l := &lexer{
		name:  name,
		input: input,
		state: scanToken,
		items: make(chan *token.Token, 2), // Two items sufficient.
		pos:   token.Pos{Line: 1, Column: 1},
		start: token.Pos{Line: 1, Column: 1},
	}
	return l
}

// nextItem returns the next item from the input.
func (l *lexer) Scan() *token.Token {
	for {
		select {
		case item := <-l.items:
			return item
		default:
			l.state = l.state(l)
		}
	}
}
