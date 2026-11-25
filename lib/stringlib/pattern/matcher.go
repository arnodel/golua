package pattern

type patternMatcher struct {
	Pattern
	s          string // string to match
	captures   [10]Capture
	si         int // current index in s (string to match)
	ci         int
	pi         int // current index in pattern
	trackbacks []trackback

	meter CPUMeter // What the steps are charged to, if not nil
	steps uint64   // Steps taken since the last charge
}

type trackback struct {
	si, ci, pi int
	siMin      int
}

func (m *patternMatcher) reset(si int) {
	m.trackbacks = nil
	m.si = si
	m.ci = 0
	m.pi = 0
}

func (m *patternMatcher) find() []Capture {
	// A pattern with no items and an end anchor matches at the end of the
	// string and nowhere else.  Going there directly means that every other
	// pattern tries an item, which is a step, at each position it starts
	// from: without this, a match with "$" would go through the string
	// without taking a step.
	if len(m.items) == 0 && m.endAnchor && m.si < len(m.s) {
		m.si = len(m.s)
	}
	for si := m.si; si <= len(m.s); si++ {
		m.reset(si)
		if captures := m.matchToEnd(); captures != nil {
			return captures
		}
	}
	return nil
}

func (m *patternMatcher) findFromStart() []Capture {
	if m.startAnchor {
		return m.matchToEnd()
	}
	return m.find()
}

func (m *patternMatcher) matchToEnd() []Capture {
	m.captures[0].start = m.si
	for {
		m.match()
		if m.si == -1 {
			return nil
		}
		if !m.endAnchor || m.si == len(m.s) {
			m.captures[0].end = m.si
			return m.captures[:m.captureCount+1]
		}
		m.trackback()
	}
}

func (m *patternMatcher) match() {
	for m.pi < len(m.items) {
		if countSteps {
			// Trying an item is a step, whether or not it matches anything.
			// Without this a pattern of items that match nothing costs
			// nothing, although each one is tried at every position of the
			// string.
			m.steps++
			if m.steps >= stepsPerCharge {
				m.chargeSteps()
			}
		}
		switch item := m.items[m.pi]; item.ptnType {
		case ptnOnce:
			if !m.matchNext(item.bytes) {
				m.trackback()
			} else {
				m.pi++
			}
		case ptnGreedyRepeat:
			si := m.si
			for m.matchNext(item.bytes) {
			}
			m.pi++
			if si < m.si {
				m.addTrackback(si)
			}
		case ptnGreedyRepeatOnce:
			if !m.matchNext(item.bytes) {
				m.trackback()
			} else {
				si := m.si
				for m.matchNext(item.bytes) {
				}
				m.pi++
				if si < m.si {
					m.addTrackback(si)
				}
			}
		case ptnRepeat:
			if m.matchNext(item.bytes) {
				m.addTrackback(m.si)
				m.si--
			}
			m.pi++
		case ptnOptional:
			si := m.si
			m.pi++
			if m.matchNext(item.bytes) {
				m.addTrackback(si)
			}
		case ptnCapture:
			c := m.captures[item.bytes[0]]
			end := m.si + c.end - c.start
			// A position capture has no text (its end is -1), and nothing
			// matches a reference to it.  Without this test its bounds would
			// be used to slice the string.
			if c.end < 0 || end > len(m.s) {
				m.trackback()
				break
			}
			if countSteps {
				// Comparing with the capture takes a step per byte.
				m.steps += uint64(c.end - c.start)
			}
			if m.s[c.start:c.end] == m.s[m.si:end] {
				m.si = end
				m.pi++
			} else {
				m.trackback()
			}
		case ptnBalanced:
			op := byte(item.bytes[0])
			if b, ok := m.getNext(); !ok || b != op {
				m.trackback()
			} else {
				cl := byte(item.bytes[1])
				depth := 1
			BLoop:
				for {
					b, ok := m.getNext()
					if !ok {
						m.trackback()
						break BLoop
					}
					switch b {
					case cl:
						depth--
						if depth == 0 {
							m.pi++
							break BLoop
						}
					case op:
						depth++
					}
				}
			}
		case ptnFrontier:
			var p, n byte
			if m.si > 0 {
				p = m.s[m.si-1]
			}
			if m.si < len(m.s) {
				n = m.s[m.si]
			}
			s := item.bytes
			if s.contains(p) || !s.contains(n) {
				m.trackback()
			} else {
				m.pi++
			}
		case ptnStartCapture:
			// The end of the capture is set to -1.  If this is an empty
			// capture, no ptnEndCapture item was emitted so the end will remain
			// -1.
			m.captures[item.bytes[0]] = Capture{m.si, -1}
			m.pi++
		case ptnEndCapture:
			m.captures[item.bytes[0]].end = m.si
			m.pi++
		default:
			panic("???")
		}
	}
}

func (m *patternMatcher) matchNext(s byteSet) bool {
	match := m.si < len(m.s) && s.contains(m.s[m.si])
	if match {
		m.si++
		if countSteps {
			m.steps++
		}
	}
	return match
}

func (m *patternMatcher) getNext() (b byte, ok bool) {
	ok = m.si < len(m.s)
	if ok {
		b = m.s[m.si]
		m.si++
		if countSteps {
			m.steps++
		}
	}
	return
}

func (m *patternMatcher) trackback() {
	i := len(m.trackbacks) - 1
	if i < 0 {
		m.pi = len(m.items)
		m.si = -1
	} else {
		t := &m.trackbacks[i]
		m.si = t.si
		m.pi = t.pi
		m.ci = t.ci
		if t.si > t.siMin {
			t.si--
		} else {
			m.trackbacks = m.trackbacks[:i]
		}
	}
}

func (m *patternMatcher) addTrackback(siMin int) {
	m.trackbacks = append(m.trackbacks, trackback{m.si, m.ci, m.pi, siMin})
}

// stepsPerCharge is how many steps a match takes before it charges them.
const stepsPerCharge = 1000

// chargeSteps charges the steps taken since the last charge.
//
// A step is an item tried, a character taken by an item, or a byte compared
// with a capture: all the work of a match is a bounded number of operations
// for each step.
//
// The steps are counted where they are taken, but the count is only tested
// once per pattern item, in match.  A call where the steps are counted would
// stop matchNext and getNext being inlined, which makes long matches about
// 40% slower.  So the steps not charged yet can be more than stepsPerCharge,
// by as many as one item can take: the length of the string.
//
//go:noinline
func (m *patternMatcher) chargeSteps() {
	steps := m.steps
	m.steps = 0
	if m.meter != nil {
		m.meter.RequireCPU(steps)
	}
}
