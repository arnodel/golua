package ir

// PreloadConstants gives the constants a function loads registers of their
// own, which hold them from the start of every call: the function's register
// template (see RegData.Preloaded), which the code generator lays out and
// the runtime copies into each new frame.  The code no longer loads them.
// The constant in
//
//	local y = x + 1          r6 <- 1; r6 <- r5 + r6
//
// is loaded each time the line runs; with a preloaded register r7 holding 1,
// it becomes
//
//	r6 <- r5 + r7
//
// A load is removed when its destination is a temporary that nothing else
// writes, so that every read of it reads this constant; the reads then read
// the preloaded register.  Each distinct constant gets one preloaded
// register, shared by all its uses in the function.  Loads into named
// locals and cells stay.
//
// Preloading pays when the frame copy, a few values, is cheaper than the
// loads it replaces, which is so in small functions called often; in a big
// function it keeps every constant alive for the whole call for little
// gain.  PreloadSmall applies it below a size.
func PreloadConstants(c Code) Code {
	writes := make([]int, len(c.Registers))
	for _, instr := range c.Instructions {
		for _, r := range instr.Writes() {
			writes[r]++
		}
	}
	e := NewEdits(c)
	preloaded := map[uint]Register{} // constant -> its register
	for i, instr := range c.Instructions {
		load, ok := instr.(LoadConst)
		if !ok {
			continue
		}
		if data := c.Registers[load.Dst]; data.IsCell || data.IsLocal || writes[load.Dst] != 1 {
			continue
		}
		r, ok := preloaded[load.Kidx]
		if !ok {
			r = e.NewRegister(RegData{Preloaded: true, Kidx: load.Kidx})
			preloaded[load.Kidx] = r
		}
		e.Delete(i)
		e.RenameReads(load.Dst, r)
	}
	if len(preloaded) == 0 {
		return c
	}
	return e.Apply(c)
}

// PreloadSmall is PreloadConstants for functions of at most maxLen
// instructions, and leaves bigger ones alone.
func PreloadSmall(maxLen int) Pass {
	return func(c Code) Code {
		if len(c.Instructions) > maxLen {
			return c
		}
		return PreloadConstants(c)
	}
}
