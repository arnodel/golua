package pattern

type patternBuilder struct {
	items  []patternItem
	ciMax  uint64
	cStack []uint64
	ptn    string
	i      int
}

func (pb *patternBuilder) getPattern() (*Pattern, error) {
	for pb.i < len(pb.ptn) {
		err := pb.getPatternItem()
		if err != nil {
			return nil, err
		}
	}
	if len(pb.cStack) != 0 {
		return nil, errInvalidPattern
	}
	return &Pattern{
		items:        pb.items,
		captureCount: int(pb.ciMax),
	}, nil
}

func (pb *patternBuilder) next() (byte, error) {
	if pb.i >= len(pb.ptn) {
		return 0, errInvalidPattern
	}
	b := pb.ptn[pb.i]
	pb.i++
	return b, nil
}

func (pb *patternBuilder) back() {
	pb.i--
}

func (pb *patternBuilder) emit(item patternItem) {
	pb.items = append(pb.items, item)
}

func (pb *patternBuilder) getPatternItem() error {
	b, err := pb.next()
	if err != nil {
		return err
	}
	var s byteSet
	switch b {
	case '(':
		pb.ciMax++
		pb.cStack = append(pb.cStack, pb.ciMax)
		if pb.ciMax >= 10 {
			return errInvalidPattern
		}
		pb.emit(patternItem{byteSet{pb.ciMax}, ptnStartCapture})
		return nil
	case ')':
		i := len(pb.cStack) - 1
		if i < 0 {
			return errInvalidPattern
		}
		pb.emit(patternItem{byteSet{pb.cStack[i]}, ptnEndCapture})
		pb.cStack = pb.cStack[:i]
		return nil
	case '%':
		c, err := pb.next()
		if err != nil {
			return err
		}
		switch {
		case c == 'f':
			s, err := pb.getCharClass()
			if err == nil {
				pb.emit(patternItem{s, ptnFrontier})
			}
			return err
		case c == 'b':
			op, err := pb.next()
			if err != nil {
				return err
			}
			cl, err := pb.next()
			if err != nil {
				return err
			}
			if op == cl {
				return errInvalidPattern
			}
			pb.emit(patternItem{[4]uint64{uint64(op), uint64(cl)}, ptnBalanced})
			return nil
		case c >= '1' && c <= '9':
			ci := uint64(c - '0')
			pb.emit(patternItem{[4]uint64{ci}, ptnCapture})
			return nil
		default:
			s = getCharRange(c)
		}
	default:
		pb.back()
		s, err = pb.getCharClass()
		if err != nil {
			return err
		}
	}
	b, err = pb.next()
	ptnType := ptnOnce
	if err == nil {
		switch b {
		case '*':
			ptnType = ptnGreedyRepeat
		case '+':
			ptnType = ptnGreedyRepeatOnce
		case '-':
			ptnType = ptnRepeat
		case '?':
			ptnType = ptnOptional
		default:
			pb.back()
		}
	}
	pb.emit(patternItem{s, ptnType})
	return nil
}

func (pb *patternBuilder) getCharClass() (byteSet, error) {
	b, err := pb.next()
	if err != nil {
		return byteSet{}, err
	}
	switch b {
	case '.':
		return fullSet, nil
	case '%':
		b, err := pb.next()
		if err != nil {
			return byteSet{}, err
		}
		return getCharRange(b), nil
	case '[':
		return pb.getUnion()
	default:
		s := byteSet{}
		s.add(b)
		return s, nil
	}
}

func (pb *patternBuilder) getUnion() (s byteSet, err error) {
	var b byte
	b, err = pb.next()
	neg := false
	switch b {
	case ']':
		s.add(b)
		b, err = pb.next()
	case '^':
		neg = true
		b, err = pb.next()
	}
Loop:
	for err == nil {
		switch {
		case b == ']':
			if neg {
				s.complement()
			}
			return
		case b == '%':
			b, err = pb.next()
			if err != nil {
				return
			}
			s.merge(getCharRange(b))
		default:
			c := b
			b, err = pb.next()
			if err != nil {
				return
			}
			if b == '-' {
				b, err = pb.next()
				if err != nil {
					return
				}
				if b == ']' {
					s.add('-')
					continue Loop
				}
				s.merge(byteRange(c, b))
			} else {
				s.add(c)
				continue Loop
			}
		}
		b, err = pb.next()
	}
	return
}

func getCharRange(c byte) byteSet {
	s, ok := namedByteSet[c]
	if !ok {
		s.add(c)
	}
	return s
}
