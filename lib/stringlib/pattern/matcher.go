package pattern

type patternMatcher struct {
	Pattern
	s          string // string to match
	captures   [10]Capture
	si         int // current index in s (string to match)
	ci         int
	pi         int // current index in pattern
	trackbacks []trackback

	budget uint64
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
			if end <= len(m.s) && m.s[c.start:c.end] == m.s[m.si:end] {
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
		m.consumeBudget()
	}
	return match
}

func (m *patternMatcher) getNext() (b byte, ok bool) {
	ok = m.si < len(m.s)
	if ok {
		b = m.s[m.si]
		m.si++
		m.consumeBudget()
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

func (m *patternMatcher) consumeBudget() {
	if m.budget == 0 {
		return
	}
	m.budget--
	if m.budget == 0 {
		panic(budgetConsumed)
	}
}

var budgetConsumed interface{} = "budget consumed"
