-- A captured variable that nothing writes once a closure can see it is
-- "fixed" (ir.markFixed), and copy propagation reads it where it is used
-- rather than copying it first.  These are the cases where a captured
-- variable is not fixed, or where reading it late must still give the same
-- value, which must behave as before.

-- A closure writes the variable: the first n is read before inc runs.
local n = 0
local function inc() n = n + 1 end
print(n, inc(), n)
--> =0	nil	1

-- A grandchild closure writes a grandparent's local.
local a = 1
local function outer() return function() a = a + 1 end end
local bump = outer()
print(a, bump(), a)
--> =1	nil	2

-- A __close handler writes the variable.
local v = 1
local function closing()
  local c <close> = setmetatable({}, {__close = function() v = 2 end})
end
print(v, closing(), v)
--> =1	nil	2

-- _ENV reassigned from a nested function: the first x is the old one.
x = 1
local function setenv() _ENV = setmetatable({x = 2}, {__index = _G}) end
print(x, setenv(), x)
--> =1	nil	2

-- A loop local captured each time round is a new variable each time.
local fs = {}
for i = 1, 3 do
  local y = i * 10
  fs[i] = function() return y end
  print(y, fs[i](), y)
end
--> =10	10	10
--> =20	20	20
--> =30	30	30

-- A fixed variable: written before any closure sees it, then read late.
local z
if n > 0 then z = "set" else z = "unset" end
local function readz() return z end
print(z, readz(), z)
--> =set	set	set
