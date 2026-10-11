package base_test

import (
	"bytes"
	"fmt"
	"sync"
	"testing"

	"github.com/arnodel/golua/lib"
	rt "github.com/arnodel/golua/runtime"
)

// TestConcurrentRuntimes creates and runs runtimes in parallel. Loading the
// base library must not write to state shared between runtimes (the next and
// ipairs iterator functions are package-level values). Run with -race.
func TestConcurrentRuntimes(t *testing.T) {
	const n = 16
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var out bytes.Buffer
			r := rt.New(&out)
			cleanup := lib.LoadAll(r)
			defer cleanup()
			chunk, err := r.CompileAndLoadLuaChunk("test", []byte(`
				local n = 0
				for _ = 1, 50 do
					for _, v in ipairs({1, 2, 3}) do n = n + v end
					for k in pairs({a = 1, b = 2}) do n = n + #k end
					local k, v = next({x = 1})
					n = n + v
				end
				return n`), rt.TableValue(r.GlobalEnv()))
			if err != nil {
				errs <- err
				return
			}
			v, err := rt.Call1(r.MainThread(), rt.FunctionValue(chunk))
			if err != nil {
				errs <- err
				return
			}
			if n, ok := v.TryInt(); !ok || n != 450 {
				errs <- fmt.Errorf("script returned %v, want 450", v)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
