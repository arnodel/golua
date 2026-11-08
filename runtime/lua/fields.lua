-- A constant-key field access remembers the slot that answered last time
-- and checks it first.  These check that the guess never wins wrongly.

-- One site sees tables of different layouts.
local function get(t) return t.x end
local function set(t, v) t.x = v end
local a = {x = 1, y = 2}
local b = {y = 3, x = 4}
local c = {z = 5, y = 6, x = 7, w = 8}
print(get(a), get(b), get(c), get({}), get({y = 9}))
--> =1	4	7	nil	nil

set(a, 10) set(b, 11) set(c, 12)
print(get(a), get(b), get(c))
--> =10	11	12

-- The key moves when the table grows, and goes when it is removed.
local t = {x = 1}
for i = 1, 20 do t["k" .. i] = i end
print(get(t))
--> =1
set(t, nil)
print(get(t), t.k20)
--> =nil	20
set(t, 2)
print(get(t))
--> =2

-- A key found through __index is not remembered as the table's own.
local class = {x = "class"}
local obj = setmetatable({}, {__index = class})
print(get(obj), get(class))
--> =class	class
set(obj, "own")
print(get(obj), get(class))
--> =own	class
set(obj, nil)
print(get(obj))
--> =class

-- Storing into a key that is absent goes through __newindex.
local log = {}
local guarded = setmetatable({x = 1}, {__newindex = function(t, k, v) log[#log + 1] = k end})
set(guarded, 2)
print(guarded.x, #log)
--> =2	0
guarded.x = nil
set(guarded, 3)
print(rawget(guarded, "x"), log[1])
--> =nil	x

-- The same site, a table and then a number.
local function len(v) return v.len end
print(len({len = 3}), (pcall(len, 4)))
--> =3	false

-- A method found through the metatable's __index: the site remembers where
-- __index is in the metatable and where the method is in the class.
local A = {}
A.__index = A
function A.who(self) return "A" .. self.n end
local B = {}
B.__index = B
function B.who(self) return "B" .. self.n end
local function who(o) return o:who() end
local a, b = setmetatable({n = 1}, A), setmetatable({n = 2}, B)
print(who(a), who(b), who(a))
--> =A1	B2	A1

-- The class changes: a method replaced, removed, found deeper in a chain.
function A.who(self) return "A'" .. self.n end
print(who(a))
--> =A'1
A.who = nil
local Base = {who = function(self) return "base" .. self.n end}
setmetatable(A, {__index = Base})
print(who(a))
--> =base1

-- __index is a function, or the metatable is swapped or dropped.
setmetatable(a, {__index = function(t, k) return k .. "?" end})
print(a.who, (pcall(who, a)))
--> =who?	false
setmetatable(a, nil)
print(pcall(who, a))
--> ~^false\t.*attempt to call a nil value

-- An instance field shadows the class's.
local c = setmetatable({n = 3, who = function() return "own" end}, B)
print(who(c), who(b))
--> =own	B2
