//go:build !nocontpool
// +build !nocontpool

package runtime

// Size of the GoCont pool. The value of 10 was reached by trial-and-error but
// is probably not optimal.
const goContPoolSize = 10

// Pool for reusing Go continuations. It is modelled exactly on luacontpool.go,
// which has some documentation :)
type goContPool struct {
	conts [goContPoolSize]*GoCont
	next  int
}

func (p *goContPool) get() *GoCont {
	if p.next == 0 {
		return new(GoCont)
	}
	p.next--
	c := p.conts[p.next]
	p.conts[p.next] = nil
	return c
}

func (p *goContPool) release(c *GoCont) {
	*c = GoCont{}
	if p.next == goContPoolSize {
		return
	}
	p.conts[p.next] = c
	p.next++
}
