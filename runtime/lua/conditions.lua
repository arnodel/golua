-- Conditions in if, elseif, while and repeat are compiled to jumps, not to a
-- value that is then tested.  They must agree with the same expression
-- computed as a value, for every combination of operand values.

local shapes = {
    "a", "not a", "not not a",
    "a and b", "a or b", "not (a and b)", "not (a or b)",
    "a and b and c", "a or b or c",
    "a or b and c", "(a or b) and c", "a and b or c", "a and (b or c)",
    "not a and b", "a and not b or c", "not (a or b) and c",
    "not (not a or not (b and c))",
    "a == b", "a ~= b", "not (a ~= b)", "a ~= b and b ~= c", "a == b or b ~= c",
}

local template = [[
local a, b, c = ...
local value = not not (%s)
local r_if
if %s then r_if = true else r_if = false end
local r_elseif
if false then r_elseif = 0 elseif %s then r_elseif = true else r_elseif = false end
local n = 0
while %s do n = n + 1; break end
local k = 0
repeat
    k = k + 1
    if k > 1 then break end
until %s
return value, r_if, r_elseif, n == 1, k == 1
]]

local vals = {n = 5, nil, false, true, 0, "x"}
local failures = 0
for _, shape in ipairs(shapes) do
    local f = assert(load(template:format(shape, shape, shape, shape, shape)))
    for i = 1, vals.n do
        for j = 1, vals.n do
            for k = 1, vals.n do
                local a, b, c = vals[i], vals[j], vals[k]
                local value, r_if, r_elseif, r_while, r_repeat = f(a, b, c)
                if r_if ~= value or r_elseif ~= value or r_while ~= value or r_repeat ~= value then
                    failures = failures + 1
                    print(shape, a, b, c, value, r_if, r_elseif, r_while, r_repeat)
                end
            end
        end
    end
end
print(failures)
--> =0

-- Operands are evaluated left to right, and only until the result is known.
local log
local function v(name, x)
    log = log .. name
    return x
end

log = ""
if v("a", false) or v("b", true) and v("c", nil) then log = log .. "!" end
print(log)
--> =abc

log = ""
if v("a", 1) or v("b", true) and v("c", nil) then log = log .. "!" end
print(log)
--> =a!

log = ""
if not (v("a", 1) and v("b", false)) then log = log .. "!" end
print(log)
--> =ab!

log = ""
local n = 0
while v("a", n < 2) and v("b", true) do n = n + 1 end
print(log)
--> =ababa

log = ""
repeat local x = v("a", 1) until v("b", x == 1) or v("c", true)
print(log)
--> =ab

-- Constant conditions.
local count = 0
while true do
    count = count + 1
    if count == 3 then break end
end
print(count)
--> =3

if nil then print("nil") elseif false then print("false") elseif 0 then print("0") end
--> =0

if "" and not nil then print("yes") end
--> =yes

repeat count = count + 1 until 1
print(count)
--> =4

-- The repeat condition can use the body's locals, also when they are
-- captured.
local fs = {}
local i = 0
repeat
    i = i + 1
    local j = i
    fs[i] = function() return j end
until j >= 3 and fs[j]() == j
print(#fs, fs[1](), fs[3]())
--> =3	1	3

-- Loops on a constant condition
do
    local n = 0
    repeat n = n + 1; if n == 3 then break end until false
    repeat n = n + 10 until true
    repeat n = n + 100 until not nil
    print(n)
end
--> =113
