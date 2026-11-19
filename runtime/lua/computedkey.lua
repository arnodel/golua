-- A site that indexes with a string it computed remembers where the key
-- was, as a site with the key written in the source does.  The key changes
-- from one access to the next here, so what it remembers is often wrong.

local t = {a = 1, b = 2, c = 3}
local out = {}
for i, k in ipairs{"a", "b", "c", "a", "a", "zz", "b"} do
    out[i] = tostring(t[k])
end
print(table.concat(out, " "))
--> =1 2 3 1 1 nil 2

-- Stores, to keys present and absent.
for _, k in ipairs{"a", "d", "a", "d", "e"} do
    t[k] = (t[k] or 0) + 10
end
print(t.a, t.d, t.e, t.b)
--> =21	20	10	2

-- A key removed and set again.
for _, v in ipairs{false, 5, false, 6} do
    local k = "c"
    t[k] = v or nil
    out[#out + 1] = tostring(t[k])
end
print(out[8], out[9], out[10], out[11], t.c)
--> =nil	5	nil	6	6

-- Different tables at one site: with the key, inheriting it, without it.
local base = {x = "inherited"}
local objs = {{x = 1}, setmetatable({}, {__index = base}), {y = 2}, {x = 4}}
local k = "x"
for i = 1, #objs do
    out[i] = tostring(objs[i][k])
end
print(out[1], out[2], out[3], out[4])
--> =1	inherited	nil	4

-- __newindex is asked for a key that has no value, and only then.
local log = {}
local p = setmetatable({}, {__newindex = function(t, k, v)
    log[#log + 1] = k
    rawset(t, k, v)
end})
for _, k in ipairs{"a", "b", "a", "b"} do
    p[k] = (rawget(p, k) or 0) + 1
end
print(#log, log[1], log[2], p.a, p.b)
--> =2	a	b	2	2

-- Keys of other types at the same site.
local mixed = {"one", x = "ex", [true] = "yes", [2.5] = "float"}
for i, k in ipairs{1, "x", true, 2.5, "x", 1.0} do
    out[i] = mixed[k]
end
print(out[1], out[2], out[3], out[4], out[5], out[6])
--> =one	ex	yes	float	ex	one

-- A value that is not a table.
local s = "hello"
local m = "upper"
print(s[m](s), pcall(function() local n = nil; return n[m] end))
--> ~HELLO\tfalse\t.*attempt to index a nil value
