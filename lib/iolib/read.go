package iolib

import (
	"errors"

	rt "github.com/arnodel/golua/runtime"
)

func ioread(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	next := c.Next()
	readers, fmtErr := getFormatReaders(c.Etc())
	if fmtErr != nil {
		return nil, rt.NewErrorE(fmtErr).AddContext(c)
	}
	err := read(t.Runtime, getIoData(t).defaultInputFile(), readers, next)
	if err != nil {
		return nil, err.AddContext(c)
	}
	return next, nil
}

func fileread(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	f, err := FileArg(c, 0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	next := c.Next()
	readers, fmtErr := getFormatReaders(c.Etc())
	if fmtErr != nil {
		return nil, rt.NewErrorE(fmtErr).AddContext(c)
	}
	err = read(t.Runtime, f, readers, next)
	if err != nil {
		return nil, err.AddContext(c)
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
		switch s[0] {
		case 'n':
			reader = (*File).ReadNumber
		case 'a':
			reader = (*File).ReadAll
		case 'l':
			reader = lineReader(false)
		case 'L':
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

func read(r *rt.Runtime, f *File, readers []formatReader, next rt.Cont) *rt.Error {
	if len(readers) == 0 {
		readers = []formatReader{lineReader(false)}
	}
	for _, reader := range readers {
		val, readErr := reader(f)
		r.Push1(next, val)
		if readErr != nil {
			break
		}
	}
	return nil
}

func lineReader(withEnd bool) formatReader {
	return func(f *File) (rt.Value, error) {
		return f.ReadLine(withEnd)
	}
}
