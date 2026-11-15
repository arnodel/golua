package runtime

import (
	"errors"
	"testing"
)

// terminations counts the terminations chained from c through their parents.
func terminations(c Cont) int {
	n := 0
	for {
		term, ok := c.(*Termination)
		if !ok {
			return n
		}
		n++
		c = term.parent
	}
}

// A call from Go into Lua receives its results in a termination whose parent
// is the thread's current continuation.  The call must leave the current
// continuation as it found it: if it left its own termination there, the next
// call's termination would have that one as its parent, and so on, and every
// one of them would stay reachable until the function running them returns.
func TestCallFromGoRestoresCurrentCont(t *testing.T) {
	r := New(nil)
	env := r.GlobalEnv()

	// probe records how many terminations are chained from the one waiting
	// for its result.
	longest := 0
	probe := func(t *Thread, c *GoCont) (Cont, error) {
		if n := terminations(c.Next()); n > longest {
			longest = n
		}
		return c.PushingNext1(t.Runtime, IntValue(1)), nil
	}
	probeFn := r.SetEnvGoFunc(env, "probe", probe, 2, false)

	t.Run("metamethods called by one run of a function", func(t *testing.T) {
		// o.a calls probe as the __index metamethod, a hundred times with
		// no Lua call between them.
		meta := NewTable()
		r.SetTable(meta, StringValue("__index"), FunctionValue(probeFn))
		o := NewTable()
		o.SetMetatable(meta)
		r.SetTable(env, StringValue("o"), TableValue(o))
		longest = 0
		got := runChunk(t, r, `local s = 0 for i = 1, 100 do s = s + o.a end return s`)
		if got != IntValue(100) {
			t.Fatalf("got %v, want 100", got)
		}
		if longest != 1 {
			t.Errorf("%d terminations chained, want 1", longest)
		}
	})

	t.Run("calls from the host program", func(t *testing.T) {
		longest = 0
		for i := 0; i < 100; i++ {
			if _, err := Call1(r.MainThread(), FunctionValue(probeFn)); err != nil {
				t.Fatal(err)
			}
		}
		if longest != 1 {
			t.Errorf("%d terminations chained, want 1", longest)
		}
	})

	t.Run("calls from the host program that fail", func(t *testing.T) {
		fail := r.SetEnvGoFunc(env, "fail", func(t *Thread, c *GoCont) (Cont, error) {
			if n := terminations(c.Next()); n > longest {
				longest = n
			}
			return nil, errors.New("failed")
		}, 0, false)
		longest = 0
		for i := 0; i < 100; i++ {
			if _, err := Call1(r.MainThread(), FunctionValue(fail)); err == nil {
				t.Fatal("expected an error")
			}
		}
		if longest != 1 {
			t.Errorf("%d terminations chained, want 1", longest)
		}
	})

	t.Run("a Go function that calls Lua", func(t *testing.T) {
		var before, after Cont
		r.SetEnvGoFunc(env, "callback", func(t *Thread, c *GoCont) (Cont, error) {
			before = t.CurrentCont()
			if _, err := Call1(t, c.Arg(0)); err != nil {
				return nil, err
			}
			after = t.CurrentCont()
			return c.Next(), nil
		}, 1, false)
		runChunk(t, r, `callback(function() return 1 end)`)
		if before == nil || before != after {
			t.Errorf("current continuation is %v after the call, was %v", after, before)
		}
	})
}
