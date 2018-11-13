package runtime

import (
	"encoding/binary"
	"io"
)

// WriteConst serializes a const value to the writer w.
func WriteConst(w io.Writer, c Konst) (err error) {
	if c == nil {
		_, err = w.Write([]byte{constTypeNil})
	} else {
		err = c.writeKonst(w)
	}
	return
}

// LoadConst reads from r to deserialize a const value.
func LoadConst(r io.Reader) (Konst, error) {
	var tp = make([]byte, 1)
	_, err := r.Read(tp)
	if err != nil {
		return nil, err
	}
	switch tp[0] {
	case constTypeInt:
		var x int64
		if err := bread(r, &x); err != nil {
			return nil, err
		}
		return Int(x), nil
	case constTypeFloat:
		var x float64
		if err := bread(r, &x); err != nil {
			return nil, err
		}
		return Float(x), nil
	case constTypeString:
		var l uint64
		if err := bread(r, &l); err != nil {
			return nil, err
		}
		var b = make([]byte, l)
		if _, err := r.Read(b); err != nil {
			return nil, err
		}
		return String(b), nil
	case constTypeBool:
		var x bool
		if err := bread(r, &x); err != nil {
			return nil, err
		}
		return Bool(x), nil
	case constTypeNil:
		return nil, nil
	case constTypeCode:
		x := new(Code)
		if err := x.loadKonst(r); err != nil {
			return nil, err
		}
		return x, nil
	}
	return nil, nil
}

// Konst is a runtime value that is a constant
type Konst interface {
	Value
	writeKonst(io.Writer) error
}

func (n Int) writeKonst(w io.Writer) error {
	_, err := w.Write([]byte{constTypeInt})
	if err == nil {
		err = bwrite(w, int64(n))
	}
	return err
}

func (f Float) writeKonst(w io.Writer) error {
	_, err := w.Write([]byte{constTypeFloat})
	if err == nil {
		err = bwrite(w, float64(f))
	}
	return err
}

func swrite(w io.Writer, s string) (err error) {
	err = bwrite(w, int64(len(s)))
	if err == nil {
		err = bwrite(w, []byte(s))
	}
	return
}

func sread(r io.Reader, s *string) (err error) {
	var sl int64
	err = bread(r, &sl)
	if err == nil {
		b := make([]byte, sl)
		err = bread(r, b)
		if err == nil {
			*s = string(b)
		}
	}
	return
}

func (s String) writeKonst(w io.Writer) error {
	_, err := w.Write([]byte{constTypeString})
	if err == nil {
		err = swrite(w, string(s))
	}
	return err
}

func (b Bool) writeKonst(w io.Writer) error {
	_, err := w.Write([]byte{constTypeBool})
	if err == nil {
		err = bwrite(w, bool(b))
	}
	return err
}

const (
	constTypeInt = iota
	constTypeFloat
	constTypeString
	constTypeBool
	constTypeNil
	constTypeCode

	// ConstTypeMaj is bigger than any const type
	ConstTypeMaj
)

func bwrite(w io.Writer, x interface{}) (err error) {
	err = binary.Write(w, binary.LittleEndian, x)
	return
}

func bread(r io.Reader, x interface{}) error {
	return binary.Read(r, binary.LittleEndian, x)
}
