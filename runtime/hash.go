package runtime

import "unsafe"

// Previously we used the following
//
// //go:linkname goRuntimeInt64Hash runtime.int64Hash //go:noescape func
// goRuntimeInt64Hash(i uint64, seed uintptr) uintptr
//
// //go:linkname goRuntimeEfaceHash runtime.efaceHash //go:noescape func
// goRuntimeEfaceHash(i interface{}, seed uintptr) uintptr
//
// But since go 1.23 it is no longer allowed to use //go.linkname to refer to
// internal symbols in the standard library (see
// https://tip.golang.org/doc/go1.23#linker).
//
// This means the above is no longer possible - fortunately, these functions are
// implemented in Go and the functions they call are all exceptions to the rule.
// So we work around the new restriction by copying those implementations into
// our codebase.
//
// This should be fairly stable as the reasons why memhash64, nilinterhash and
// noescape are exceptions is that they are used in a number of major open
// source projects.

// The two functions below are copied from
// https://github.com/golang/go/blob/release-branch.go1.23/src/runtime/alg.go#L446-L452

func goRuntimeInt64Hash(i uint64, seed uintptr) uintptr {
	return memhash64(noescape(unsafe.Pointer(&i)), seed)
}

func goRuntimeEfaceHash(i interface{}, seed uintptr) uintptr {
	return nilinterhash(noescape(unsafe.Pointer(&i)), seed)
}

//go:linkname memhash64 runtime.memhash64
//go:noescape
func memhash64(p unsafe.Pointer, h uintptr) uintptr

//go:linkname nilinterhash runtime.nilinterhash
//go:noescape
func nilinterhash(p unsafe.Pointer, h uintptr) uintptr

//go:linkname noescape runtime.noescape
//go:noescape
func noescape(p unsafe.Pointer) unsafe.Pointer
