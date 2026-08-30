package runtime

import "unsafe"

// This file provides the two hashing primitives used by Value.Hash():
//
//   - hashScalar hashes the 64-bit "scalar" representation of a Value (integers,
//     floats, booleans and strings of up to 7 bytes).
//   - hashInterface hashes the interface representation of a Value (everything
//     else: longer strings, tables, functions, userdata, ...).
//
// History: we originally reached into the Go runtime for both, via
//
//	//go:linkname hashScalar    runtime.int64Hash
//	//go:linkname hashInterface runtime.efaceHash
//
// on the assumption that the runtime's hashers would be the fastest option.
//
// Go 1.23 stopped allowing //go:linkname references to std-library internal
// symbols unless the definition is itself marked //go:linkname (see
// https://tip.golang.org/doc/go1.23#linker). runtime.int64Hash / efaceHash are
// not marked, so we switched to calling runtime.memhash64 and
// runtime.nilinterhash, which the linker still resolved.
//
// Go 1.27 then stopped resolving runtime.memhash64 - it is now a trivial
// wrapper around internal/runtime/maps.MemHash64 with no //go:linkname of its
// own - which broke the build with
//
//	link: github.com/arnodel/golua/runtime: invalid reference to runtime.memhash64
//
// (https://github.com/arnodel/golua/issues/128). runtime.nilinterhash still
// carries a //go:linkname on its definition and is unaffected.
//
// For the scalar case we now hash inline with the MurmurHash3 64-bit finalizer
// (fmix64). The runtime hasher was never actually a win here: memhash64 uses
// AES instructions that pay off for bulk memory, but for a single 8-byte value
// the non-inlinable call dominates and this finalizer is about twice as fast.
// It has good avalanche and bucketing behaviour (see hash_test.go) and needs no
// linkname. The interface case still delegates to runtime.nilinterhash, which
// knows how to hash arbitrary comparable Go types and is not something we want
// to reimplement.

// Both functions below take a seed argument. It is always 0 for now - golua
// tables have no hash-flooding protection - but is kept so that seeding the
// hashes with a random value at startup stays a local change.

// hashScalar returns a hash of the given 64-bit value, seeded with seed. The
// body is the MurmurHash3 64-bit finalizer (fmix64) by Austin Appleby, with the
// seed folded into the input. fmix64 is a bijective bit-mixer built from xorshift
// and odd-constant multiplies; it is not cryptographic but has full avalanche,
// which is all we need for hash-table bucketing. The two magic constants and the
// 33-bit shifts are from the reference implementation:
// https://github.com/aappleby/smhasher/blob/master/src/MurmurHash3.cpp (fmix64).
func hashScalar(x uint64, seed uintptr) uintptr {
	x ^= uint64(seed)
	x ^= x >> 33
	x *= 0xff51afd7ed558ccd
	x ^= x >> 33
	x *= 0xc4ceb9fe1a85ec53
	x ^= x >> 33
	return uintptr(x)
}

// hashInterface returns a hash of the given interface value, seeded with seed.
// It delegates to runtime.nilinterhash which - unlike runtime.memhash64 - is
// explicitly linknameable (widely used packages depend on it).
func hashInterface(i interface{}, seed uintptr) uintptr {
	return nilinterhash(noescape(unsafe.Pointer(&i)), seed)
}

//go:linkname nilinterhash runtime.nilinterhash
//go:noescape
func nilinterhash(p unsafe.Pointer, h uintptr) uintptr

//go:linkname noescape runtime.noescape
//go:noescape
func noescape(p unsafe.Pointer) unsafe.Pointer
