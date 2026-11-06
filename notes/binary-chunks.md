# Binary chunks: what makes loading them safe

`string.dump` turns a function into a binary chunk, and `load(s, nil, "b")`
(also `loadfile`, `dofile` and `require`) turns one back into a function.  A
binary chunk is bytecode that skips the compiler, so it can be anything.  This
note says what that puts at risk, the ways of dealing with it, and what golua
does.

## What a binary chunk can do

Everything a runtime context restricts is enforced by the VM and the library,
whatever the bytecode:

- CPU is charged for each opcode the run loop executes;
- memory is charged where it is allocated;
- a Go function only runs if it complies with the context's required flags;
- files are only reached through `safeio`, within the context's `FSAccess`.

None of this depends on where the bytecode came from.  A forged chunk cannot
do anything that source code couldn't, except one thing: **break the
invariants the VM relies on the compiler to guarantee.**  Breaking them crashes
the VM, with a Go panic, often on a coroutine's goroutine where nothing can
recover it, so the whole program stops.

The invariants include, at the time of writing:

- register, cell and constant operands are within the frame and the constant
  table, jumps land inside the code, and every opcode has a line (checked by
  `Code.verify`; the run loop accesses registers without bounds checks on the
  strength of it);
- registers hold the kind of value their opcode needs: a continuation for
  `push`, `call` and `return`, a varargs slice for `...`, a closure for
  `upval`, a table for a table constructor, code for a closure constant (not
  checked: fuzzing forged chunks found crashes here).

The second list grows with optimisations: each one that relies on something
the compiler guarantees adds an invariant.

## Three ways to know a chunk keeps the invariants

| | how | runtime cost | load cost | as the VM gets faster |
|---|---|---|---|---|
| **validate** | prove the invariants hold | none | an analysis of every function, compiled ones too | every new invariant needs a new rule in the verifier, and a missing rule is a hole |
| **check while running** | test each assumption where it is used | +0.21% geomean, up to 1% on call-heavy benchmarks (measured on the register kinds above) | none | each new assumption adds a check: it rules out the optimisations that rely on the compiler |
| **sign** | prove the chunk came from golua's compiler | none | an HMAC over the chunk | nothing to do: whatever the compiler guarantees, signed code keeps |

## What golua does

**Signing.**  `string.dump` signs a chunk with HMAC-SHA256 under the runtime's
key.  The key is random for each runtime unless the host provides one
(`rt.WithCodeKey`), which a bytecode cache across runs, or several processes
sharing code, needs.  Lua code never sees the key; it can only get signatures
through `string.dump`, which signs functions golua compiled, and a context that
can compile source gains nothing from that.

**Restricted contexts only load signed chunks.**  A context is restricted if
it requires any compliance flag, which any hard limit or restriction of file
access implies.  An unrestricted
context accepts any chunk, as now: it can already call `os.exit`, so crashing
the VM gives it nothing it didn't have.

**The format version is separate.**  A chunk starts with a magic prefix and a
format version, and `load` refuses a version it doesn't read, in any context,
before checking anything else.  This guards against honest mistakes, such as a
bytecode cache left over from an older golua, which a signature wouldn't: an
old chunk signed with the same key is still signed.

The version is a counter in a byte after the prefix, unrelated to golua's
release numbers, so a release that leaves the format alone keeps the chunks
of the one before.  It goes up at most once between two releases: the first
change to the opcodes, their encoding or the marshal format after a release
bumps it, and later changes before the next release keep it.  Released chunks
and chunks from development builds after a change are told apart; two
development builds in the same cycle may not be, which only matters for a
cache kept from one unreleased build to the next.  A test of a hash of the
opcode tables and the marshal format catches a change: it fails until the hash
is updated, and the version is bumped too if it is still the one of the last
release.

**The cheap checks stay.**  The loader reads lengths without trusting them,
allocating no more than the input holds (#190), and `Code.verify` keeps its
checks of indices, jumps and lines.  They cost little, the run loop relies on
the index checks, and they catch corrupt chunks in unrestricted contexts.
There is no analysis of what registers hold.

## Rejected

- **Checking register kinds in the run loop**: measured (above), and it would
  make every future invariant a runtime check.
- **Analysing register kinds in the verifier**: it would slow every load and
  compilation, and it must model the compiler exactly, where a gap is a crash.
- **Trusting directories** (an `x` action in `FSAccess` for directories whose
  chunks are trusted): it establishes provenance by location, which needs
  rules so that a context can't write a chunk where it, or a context it
  creates, trusts it.  Signing establishes provenance with no such rules, and
  covers the same uses.

## Work

1. The format version in the dump header, with the test of the opcode tables.
2. Signing in `string.dump`, `rt.WithCodeKey`, and restricted contexts refusing
   unsigned chunks.
3. The fuzz target for binary chunks: in a restricted context, any input that
   isn't a signed chunk is refused cleanly; the loader and verifier are also
   fuzzed on signed chunks, without running them.

Later, if wanted: signing with Ed25519 instead, so that code can be shipped
with a key that verifies chunks but can't sign them.
