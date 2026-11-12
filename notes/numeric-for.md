In Lua 5.4 the docs specify numeric for loops as working as follows

>The loop starts by evaluating once the three control expressions. Their values
>are called respectively the initial value, the limit, and the step. If the step
>is absent, it defaults to 1.

>If both the initial value and the step are integers, the loop is done with
>integers; note that the limit may not be an integer. Otherwise, the three
>values are converted to floats and the loop is done with floats. Beware of
>floating-point accuracy in this case.

>After that initialization, the loop body is repeated with the value of the
>control variable going through an arithmetic progression, starting at the
>initial value, with a common difference given by the step. A negative step
>makes a decreasing sequence; a step equal to zero raises an error. The loop
>continues while the value is less than or equal to the limit (greater than or
>equal to for a negative step). If the initial value is already greater than the
>limit (or less than, if the step is negative), the body is not executed.

>For integer loops, the control variable never wraps around; instead, the loop
>ends in case of an overflow.

This means that if a loop variable is an integer and this integer overflows /
underflows when it is incremented / decremented then the loops stops
immediately.  E.g. the following loop will stop after one iteration instead of i
wrapping around and the loop carrying on forever.

```lua
for i = math.maxinteger, 1e100 do end
```

It isn't practical to try to implement the semantics of the for loop as
described above in terms of existing opcodes.

Instead two new opcodes are introduced, each followed by a jump that the
opcode itself takes or steps over:
- `prepfor rStart, rStop, rStep`: makes sure `rStart`, `rStep`, `rStop` are all
  numbers and converts `rStart` and `rStep` to the same numeric type.  If the
  loop has no iteration, it takes the jump.
- `advfor rStart, rStop, rStep`: increments `rStart` by `rStep`, making sure
  that it doesn't wrap around if it is an integer.  If it wraps around or
  passes the limit, the loop is over; otherwise it takes the jump.

With these two new opcodes it becomes easy to compile a numeric for loop.
Assuming `r1`, `r2`,  `r3` contain the start, stop and step initial values:

```
     prepfor r1, r2, r3
     jump END
LOOP:
     r4 <- r1
     [compiled body of loop where the loop variable is r4]
     advfor r1, r2, r3
     jump LOOP
END:
```

The loop variable `r4` is a copy of `r1` that copy propagation removes unless
a closure captures it.  Until November 2026 the jumps were conditional jumps
on `r1`, which the opcodes set to nil to end the loop; taking the jump from
the for opcode itself saves one dispatch per iteration (PR #247).
