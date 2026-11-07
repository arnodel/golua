-- Constants used in loops are loaded once, before the loop.

local function count(n)
    local s = 0
    for i = 1, n do
        s = s + 1 + 2 + 3
    end
    return s
end
print(count(4))
--> =24

-- The register holding a hoisted constant is set each time the loop is
-- entered, including through a for loop with no iteration, which skips to
-- the statement after it.
local function skip(n, m)
    local s = 0
    for i = 1, n do s = s + 100 end
    repeat s = s + 7; m = m - 1 until m <= 0
    return s
end
print(skip(0, 3), skip(2, 1))
--> =21	207

-- A function with too many registers for hoisting is compiled without it.
local names = {}
for i = 1, 244 do names[i] = "a" .. i end
local list = table.concat(names, ", ")
local f = load("local " .. list .. " = ...\n" ..
    "local s = 0\n" ..
    "for i = 1, 3 do s = s + 1 + 2 + 3 + 4 + 5 + 6 + 7 + 8 + 9 + 10 end\n" ..
    "return s, " .. list)
local args = {}
for i = 1, 244 do args[i] = i end
local r = table.pack(f(table.unpack(args)))
print(r.n, r[1], r[2], r[245])
--> =245	165	1	244
