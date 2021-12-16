package iolib

import (
	"bufio"
	"bytes"
	"errors"
	"io"
)

type bufReader interface {
	io.Reader
	Reset(r io.Reader)
	Buffered() int
	Discard(int) (int, error)
	Peek(n int) ([]byte, error)
	ReadString(delim byte) (string, error)
}

type bufWriter interface {
	io.Writer
	Reset(w io.Writer)
	Flush() error
}
type nobufReader struct {
	io.Reader
}

var (
	_ bufReader = (*nobufReader)(nil)
	_ bufReader = (*bufio.Reader)(nil)
	_ bufWriter = (*nobufWriter)(nil)
	_ bufWriter = (*bufio.Writer)(nil)
)

func (u *nobufReader) Reset(r io.Reader) {
	u.Reader = r
}

func (u *nobufReader) Buffered() int {
	return 0
}

func (u *nobufReader) Discard(n int) (int, error) {
	if n > 0 {
		return 0, errors.New("nobufReader cannot discard")
	}
	return 0, nil
}

func (u *nobufReader) Peek(n int) ([]byte, error) {
	if n > 0 {
		return nil, errors.New("nobufReader cannot peek")
	}
	return nil, nil
}

func (u *nobufReader) ReadString(delim byte) (string, error) {
	return "", errors.New("unimplemented")
}

type nobufWriter struct {
	io.Writer
}

func (u *nobufWriter) Reset(w io.Writer) {
	u.Writer = w
}

func (u *nobufWriter) Flush() error {
	return nil
}

type linebufWriter struct {
	*bufio.Writer
}

func (u linebufWriter) Write(p []byte) (n int, err error) {
	for len(p) > 0 {
		i := bytes.IndexAny(p, "\n\r") + 1
		flush := i > 0
		if !flush {
			i = len(p)
		}
		m, err := u.Writer.Write(p[:i])
		n += m
		if err == nil && flush {
			err = u.Flush()
		}
		if err != nil {
			break
		}
		p = p[i:]
	}
	return n, err
}
