package ast

import (
	"fmt"
	"io"
)

// IndentWriter is an implementation of HWriter that writes AST with indentation
// to show the tree structure.
type IndentWriter struct {
	writer io.Writer
	depth  int
}

var _ HWriter = (*IndentWriter)(nil)

// NewIndentWriter returns a new pointer to IndentWriter that will write using
// the given io.Writer.
func NewIndentWriter(w io.Writer) *IndentWriter {
	return &IndentWriter{writer: w}
}

// Writef is like Printf.
func (w *IndentWriter) Writef(f string, args ...interface{}) {
	fmt.Fprintf(w.writer, f, args...)
}

// Indent increments the current indentation by one level.
func (w *IndentWriter) Indent() {
	w.depth++
}

// Dedent decrements the current indentation by one level.
func (w *IndentWriter) Dedent() {
	w.depth--
}

const spaces80 = "                                                                                "

// Next moves on to the next line and adds the necessary indentation.
func (w *IndentWriter) Next() {
	w.writer.Write([]byte("\n"))
	i := w.depth * 4
	for i > 80 {
		w.writer.Write([]byte(spaces80))
		i -= 80
	}
	w.writer.Write([]byte(spaces80)[:i])
}
