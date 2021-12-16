package iolib

import (
	"errors"
	"io"

	rt "github.com/arnodel/golua/runtime"
)

func ioread(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	next := c.Next()
	readers, fmtErr := getFormatReaders(c.Etc())
	if fmtErr != nil {
		return nil, rt.NewErrorE(fmtErr)
	}
	ioErr := read(t.Runtime, getIoData(t).defaultInputFile(), readers, next)
	if ioErr != nil && ioErr != io.EOF {
		return t.ProcessIoError(c.Next(), ioErr)
	}
	return next, nil
}

func fileread(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	f, err := FileArg(c, 0)
	if err != nil {
		return nil, err
	}
	next := c.Next()
	readers, fmtErr := getFormatReaders(c.Etc())
	if fmtErr != nil {
		return nil, rt.NewErrorE(fmtErr)
	}
	ioErr := read(t.Runtime, f, readers, next)
	if ioErr != nil && ioErr != io.EOF {
		return t.ProcessIoError(c.Next(), ioErr)
	}
	return next, nil
}

type formatReader func(*File) (rt.Value, error)

var errInvalidFormat = errors.New("invalid format")
var errFormatOutOfRange = errors.New("format out of range")

func getFormatReader(fmt rt.Value) (reader formatReader, err error) {
	if n, ok := rt.ToInt(fmt); ok {
		if n < 0 {
			return nil, errFormatOutOfRange
		}
		reader = func(f *File) (rt.Value, error) { return f.Read(int(n)) }
	} else if s, ok := fmt.TryString(); ok && len(s) > 0 {
		switch s {
		case "n", "*n":
			reader = (*File).ReadNumber
		case "a", "*a", "all":
			reader = (*File).ReadAll
		case "l", "*l":
			reader = lineReader(false)
		case "L", "*L":
			reader = lineReader(true)
		default:
			return nil, errInvalidFormat
		}
	} else {
		return nil, errInvalidFormat
	}
	return
}

func getFormatReaders(fmts []rt.Value) ([]formatReader, error) {
	readers := make([]formatReader, len(fmts))
	for i, fmt := range fmts {
		reader, err := getFormatReader(fmt)
		if err != nil {
			return nil, err
		}
		readers[i] = reader
	}
	return readers, nil
}

func read(r *rt.Runtime, f *File, readers []formatReader, next rt.Cont) error {
	if f.closed {
		return errFileAlreadyClosed
	}
	if len(readers) == 0 {
		readers = []formatReader{lineReader(false)}
	}
	for i, reader := range readers {
		val, readErr := reader(f)
		if readErr == nil {
			r.Push1(next, val)
		} else if i == 0 || readErr != io.EOF {
			return readErr
		}
	}
	return nil
}

func lineReader(withEnd bool) formatReader {
	return func(f *File) (rt.Value, error) {
		return f.ReadLine(withEnd)
	}
}
