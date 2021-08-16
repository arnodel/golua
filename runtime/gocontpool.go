package runtime

var globalGoContPool = goContPool{}

const goContPoolSize = 10

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
	if p.next == luaContPoolSize {
		return
	}
	p.conts[p.next] = c
	p.next++
}
