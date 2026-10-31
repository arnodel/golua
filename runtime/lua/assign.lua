-- Stores read non-cell locals directly instead of copying them first.  These
-- are the cases where the copy mattered, which must still behave as before.

-- A local assigned by the same statement is read before the assignment.
local i = 3
local a = {}
i, a[i] = i+1, 20
print(i, a[3], a[4])
--> =4	20	nil

-- Same with the table itself.
local o = {}
local old = o
o, o.x = {}, 1
print(old.x, o.x)
--> =1	nil

-- Swaps need both values copied.
local p, q = 1, 2
p, q = q, p
print(p, q)
--> =2	1

local t = {10, 20}
local m, n = 1, 2
t[m], t[n] = t[n], t[m]
print(t[1], t[2])
--> =20	10

-- A single local value stored directly.
local v = "value"
local s = {}
s.f = v
s[m] = v
print(s.f, s[1])
--> =value	value

-- A local captured by a closure (a cell) is still copied, so the store goes to
-- the table the statement started with even though a function called while
-- evaluating another l-value reassigns the local.  The manual leaves this
-- order undefined (reference Lua stores into the new table); this records
-- that golua's behaviour is unchanged.
local c = {}
local first = c
local function swap() c = {} return {} end
c.x, swap().y = 1, 2
print(first.x, c.x)
--> =1	nil

-- Assignment to a local from a local, including itself.
local x, y = 1, 2
x = y
x = x
print(x, y)
--> =2	2
