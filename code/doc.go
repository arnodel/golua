/*
Package code defines the bytecode that the golua runtime executes: the
instruction encoding, the constants a compiled function refers to, and the
code units that hold both.

# The machine

A Lua function compiles to a sequence of 32-bit opcodes operating on a frame
of registers.  There are two kinds of registers:

  - value registers (r0, r1, ...) hold a Value each;
  - cell registers (u0, u1, ...) hold a reference to a Value, so that a
    closure and the function that created it can share a variable.  The first
    cell registers of a frame are the function's upvalues, in order.

Each kind is numbered independently from 0 and a function declares how many
of each it needs (Code.RegCount and Code.CellCount).  Register operands are 8
bits wide, so a frame has at most 256 of each.

Register r0 always holds the continuation to return values to.  Calls are
made by building a continuation for the callee (cont / tailcont), pushing the
arguments into it one by one (push / pushetc) and jumping to it (call /
tailcall).  The callee starts with recv / recvetc opcodes that put the pushed
values into its registers.  A return is a call of r0; a tail call reuses the
caller's continuation as the callee's r0 and frees the caller's frame.

A variable number of values (the arguments of a vararg function, the results
of a call) travels as an "etc": a value holding a vector, built by recvetc,
pushed by pushetc, read by etclookup and fill.

# The encoding

Every opcode is one 32-bit word.  The top four bits select the type, and the
bit after them (F) is a flag whose meaning depends on the type.  Register
operands occupy the three low bytes (A, B, C), each with a kind bit (a, b, c)
in the second byte: 0 for a value register, 1 for a cell register.  Types
that do not use B or C put a literal or an operator there instead.

	bit:  31 30 29 28 | 27 | 26 25 24 | 23 .. 16 | 15 .. 8 | 7 .. 0
	      type        | F  | a  b  c  |    A     |    B    |   C

Type 1 is the exception: it uses only bit 31 as its prefix and takes bits
30-27 for the operator.

The field names below (A, B, C, F, X, Y, Z, J, L, M, N, D) are the ones used
in this package's accessor methods (GetA, GetF, GetX ...).

# Type 1: binary operations

	1 XXXX a b c | A | B | C          rA <- rB op rC

X selects the operator (BinOp): the arithmetic operators, the bitwise
operators, the comparisons ==, < and <= (which produce a boolean), and
concatenation.  Metamethods apply as in the language.

# Type 2: table access

	0111 F a b c | A | B | C

	F = Off    rA <- rB[rC]            (__index applies)
	F = On     rB[rC] <- rA            (__newindex applies)

# Type 8: table access by constant key

	0001 F a b 0 | A | B | K (8 bits)

	F = Off    rA <- rB[K]             (__index applies)
	F = On     rB[K] <- rA             (__newindex applies)

K is the index of the key in the constant table.  This is Type 2 for a
constant key, saving the opcode that would load the key into a register; the
compiler emits it for field access (obj.name, obj:method(), globals) when the
constant's index fits in 8 bits, and falls back to Type 2 otherwise.

# Type 3: load a 16-bit literal or a constant

	0110 F a Y Y | A | N (16 bits)

Y selects how N is read (UnOpK16):

	OpInt16     the integer int16(N)
	OpStr2      the 2-byte string whose bytes are N (little-endian)
	OpK         constant number N of the function's constant table
	OpClosureK  a new closure of the function constant number N; its
	            upvalues are then added with upval (Type 4a)

With F = Off the value goes to rA.  With F = On it is pushed into the
continuation held in rA instead.

# Type 4a: unary operations and continuations

	0101 F a b 1 | A | B | Z (8 bits)

Z selects the operation (UnOp) applied to rB, and the result goes to rA
(F = Off) or is pushed into the continuation in rA (F = On):

	OpNeg       -rB
	OpBitNot    ~rB
	OpLen       #rB
	OpTruth     rB as a boolean
	OpNot       not rB
	OpId        rB itself; with F = On this is "push rA, rB"
	OpCont      a new continuation for the closure rB, returning to the
	            current frame (its r0 is the current continuation)
	OpTailCont  a new continuation for the closure rB, returning where the
	            current frame returns to (its r0 is the current r0)

Three operations under this type do not follow the "result to rA" pattern:

	OpEtcId      push all the values of the etc rB into the continuation
	             rA ("pushetc rA, rB"; F is On)
	OpUpvalue    add the cell uB to the upvalues of the closure rA ("upval");
	             B must be a cell register
	OpCheckNotDefined  raise an error if rA[rB] is not nil (the Lua 5.5
	             "global x = v" redefinition check)

and OpMkVarargTable makes a table sharing the vector of the etc rB (the Lua
5.5 vararg table), stores it in rA and replaces rB with it.

# Type 4b: load a small constant

	0101 F a 0 0 | A | L (8 bits) | Z (8 bits)

Z selects the constant (UnOpK), L is its payload where there is one; the
value goes to rA (F = Off) or is pushed into the continuation in rA (F = On):

	OpNil       nil
	OpBool      the boolean L != 0
	OpStr0      ""
	OpStr1      the 1-byte string L
	OpTable     a new empty table
	OpCC        the current continuation (the frame itself)
	OpClear     not a load: "clr rA" resets the register; for a cell
	            register this makes a fresh cell, which is how a loop gives
	            each iteration its own variable for closures to capture

# Type 5: jumps, calls and the close stack

	0100 F a J J | A | D (16 bits)

J selects the operation (JumpOp):

	OpJump      pc <- pc + D (D is a signed offset from the jump itself)
	OpJumpIf    jump if rA is truthy (F = On) or falsy (F = Off)
	OpCall      push the D registers listed in the operand words that
	            follow (see below) into the continuation in rA, then
	            transfer control to it; F = On makes it a tail call, which
	            releases the current frame.  "call r0" is a return.
	OpClStack   F = On: "clpush rA", push rA onto the close stack (a value
	            with a __close metamethod, or nil/false); F = Off: "cltrunc
	            D", truncate the close stack to height D, calling __close
	            on the values removed.  These implement to-be-closed
	            variables.

For OpJump the register is unused.

A call with D > 0 arguments is followed by ceil(D/3) operand words.  They
are not opcodes: each lists up to three registers in the positions of A, B
and C, with their kind bits, and zero in the top two bits:

	00 x y z a b c | A | B | C

The call pushes the D registers in order and execution resumes after the
operand words.  Arguments that were pushed one by one before the call (an
etc, for instance) come first.  The flags x, y and z mark a register as a
temporary that nothing reads after the call: the call sets it to nil once it
is pushed, so that the value can be collected as soon as the callee lets go
of it.  No jump may land inside an operand word; the verifier checks it.

# Type 6: etc vectors

	0011 F a b 0 | A | B | M (8 bits)

	F = Off    rA <- etclookup(rB, M)   the M-th value (from 0) of the etc
	                                    rB, nil if there is none
	F = On     fill rA, M, rB           rA[M+i] <- i-th value of the etc rB,
	                                    for every i

# Type 7: numeric for loops

	0010 F a b c | A | B | C          A is the loop variable, B the limit,
	                                  C the step

	F = Off    prepare: check the three values are numbers, give A and C
	           the same numeric type, and set A to nil if the loop must not
	           run at all
	F = On     advance: A <- A + C, then set A to nil if the new value has
	           passed B or the addition wrapped around

In both cases the compiler follows the opcode with a conditional jump on A.

# Type 0: receiving arguments

	0000 F a 0 0 | A | 0 | 0

	F = Off    recv rA        the next value pushed goes to rA
	F = On     recvetc rA     all the remaining pushed values go to rA as an
	                          etc

These are the first opcodes of a function.  Pushing into a continuation
writes directly into the register named by the recv opcode at its program
counter; when the continuation is called, any recv opcodes that have not
received a value set their register to nil (or to an empty etc).

# Constants and code units

A Unit is a compiled chunk: one vector of opcodes, one of source lines, one
of constants.  A function is a Code constant giving its range of opcodes in
the unit and its register counts; its constant table is the unit's.  The
other constants are the Int, Float, String, Bool and NilType values that
Type 3 OpK refers to.

# Unused encodings

UnOpK declares OpInt, OpFloat and OpStrN, meant to carry a larger constant in
the following words; nothing emits or executes them.  OpCC is executed but
not emitted.  The compiler can emit OpStr0, OpStr1 and OpStr2 but currently
does not (see LoadShortString), as constant strings loaded through the
constant table proved faster.
*/
package code
