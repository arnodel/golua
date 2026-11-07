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
