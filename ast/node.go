package ast

import (
	"fmt"
	"io"
)

type Node interface {
	HWrite(w HWriter)
}

type HWriter interface {
	Writef(string, ...interface{})
	Indent()
	Dedent()
	Next()
}

type IndentWriter struct {
	writer io.Writer
	depth  int
}

func NewIndentWriter(w io.Writer) *IndentWriter {
	return &IndentWriter{writer: w}
}

func (w *IndentWriter) Writef(f string, args ...interface{}) {
	w.writer.Write([]byte(fmt.Sprintf(f, args...)))
}

func (w *IndentWriter) Indent() {
	w.depth++
}

func (w *IndentWriter) Dedent() {
	w.depth--
}

const spaces80 = "                                                                                "

func (w *IndentWriter) Next() {
	w.writer.Write([]byte("\n"))
	i := w.depth * 4
	for i > 80 {
		w.writer.Write([]byte(spaces80))
		i -= 80
	}
	w.writer.Write([]byte(spaces80)[:i])
}
