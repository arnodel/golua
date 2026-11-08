-- A table used as a metatable remembers its metamethods; any write to it
-- must be seen.

local mt = {__index = {x = 1}}
local a = setmetatable({}, mt)
local b = setmetatable({}, mt)
print(a.x, b.x)
--> =1	1

mt.__index = {x = 2}
print(a.x, b.x)
--> =2	2
rawset(mt, "__index", {x = 3})
print(a.x, b.x)
--> =3	3
mt.__index = nil
print(a.x, b.x)
--> =nil	nil

-- __newindex appears, then goes.
local log = {}
a.y = 1
mt.__newindex = function(t, k, v) log[#log + 1] = k end
a.z = 2
print(rawget(a, "y"), rawget(a, "z"), log[1])
--> =1	nil	z
mt.__newindex = nil
a.w = 3
print(rawget(a, "w"), #log)
--> =3	1

-- Comparison, length and call metamethods added after the fact.
local c = setmetatable({}, mt)
print(a == c)
--> =false
mt.__eq = function() return true end
print(a == c, a == b)
--> =true	true
mt.__len = function() return 42 end
print(#a)
--> =42
mt.__call = function(self, v) return v * 2 end
print(a(21))
--> =42

-- The metatable is replaced, and the old one changed afterwards.
local mt2 = {__index = {x = "two"}}
setmetatable(a, mt2)
mt.__index = {x = "old"}
print(a.x, b.x)
--> =two	old

-- The string metatable changes.
local smt = getmetatable("")
local lib = smt.__index
smt.__index = {shout = function(s) return string.upper(s) .. "!" end}
print(("hi"):shout())
--> =HI!
smt.__index = lib
print(("hi"):upper())
--> =HI

-- A table becomes a metatable after it was written to, and is written
-- to through a method call site.
local late = {}
late.__index = {x = "late"}
local d = setmetatable({}, late)
print(d.x)
--> =late
local function setidx(m, v) m.__index = v end
setidx(late, {x = "later"})
print(d.x)
--> =later
