-- "and" and "or" computed as a value: an operand whose value can't be the
-- result is only tested.  The result must be the same value as the
-- reference, where AND, OR and NOT are written with if, for every
-- combination of operand values and wherever the value goes.

local function AND(x, y) if x then return y end return x end
local function OR(x, y) if x then return x end return y end
local function NOT(x) if x then return false end return true end

local shapes = {
    {"a and b", function(a, b, c) return AND(a, b) end},
    {"a or b", function(a, b, c) return OR(a, b) end},
    {"a and b or c", function(a, b, c) return OR(AND(a, b), c) end},
    {"a or b and c", function(a, b, c) return OR(a, AND(b, c)) end},
    {"(a or b) and c", function(a, b, c) return AND(OR(a, b), c) end},
    {"a and (b or c)", function(a, b, c) return AND(a, OR(b, c)) end},
    {"a and b and c or b", function(a, b, c) return OR(AND(AND(a, b), c), b) end},
    {"a and b or b and c or a", function(a, b, c) return OR(OR(AND(a, b), AND(b, c)), a) end},
    {"(a or b) and (b or c)", function(a, b, c) return AND(OR(a, b), OR(b, c)) end},
    {"a and (b and c or a) or b", function(a, b, c) return OR(AND(a, OR(AND(b, c), a)), b) end},
    {"(a or b and c) and (c or a)", function(a, b, c) return AND(OR(a, AND(b, c)), OR(c, a)) end},
    {"not a and b or c", function(a, b, c) return OR(AND(NOT(a), b), c) end},
    {"a and not b or c", function(a, b, c) return OR(AND(a, NOT(b)), c) end},
    {"not (a and b) or c", function(a, b, c) return OR(NOT(AND(a, b)), c) end},
    {"a ~= b and c or a", function(a, b, c) return OR(AND(a ~= b, c), a) end},
    {"a == b or b ~= c and a", function(a, b, c) return OR(a == b, AND(b ~= c, a)) end},
    {"a and 1 or 2", function(a, b, c) return OR(AND(a, 1), 2) end},
    {"a and false or c", function(a, b, c) return OR(AND(a, false), c) end},
    {"a and nil or c", function(a, b, c) return OR(AND(a, nil), c) end},
    {"nil or a and b", function(a, b, c) return OR(nil, AND(a, b)) end},
    {"false and a or b", function(a, b, c) return OR(AND(false, a), b) end},
    {"true and a or b", function(a, b, c) return OR(AND(true, a), b) end},
    {"a or 'y' and b", function(a, b, c) return OR(a, AND("y", b)) end},
    {"(a or false) and (nil or b)", function(a, b, c) return AND(OR(a, false), OR(nil, b)) end},
}

-- The value goes into a new local, into an operand, into a table, into a
-- call's argument and into a return.
local template = [[
local a, b, c = ...
local id = function(x) return x end
local r_local = %s
local t = {%s}
local r_arg = id(%s)
local function ret(a, b, c) return %s end
local r_ret = ret(a, b, c)
a = %s
return r_local, t[1], r_arg, r_ret, a
]]

local vals = {n = 5, nil, false, true, 0, "x"}
local failures = 0
for _, shape in ipairs(shapes) do
    local src, ref = shape[1], shape[2]
    local f = assert(load(template:format(src, src, src, src, src)))
    for i = 1, vals.n do
        for j = 1, vals.n do
            for k = 1, vals.n do
                local a, b, c = vals[i], vals[j], vals[k]
                local want = ref(a, b, c)
                local got = table.pack(f(a, b, c))
                for n = 1, 5 do
                    if got[n] ~= want then
                        failures = failures + 1
                        print(src, n, a, b, c, got[n], want)
                    end
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
local x = v("a", false) and v("b", 1) or v("c", 2)
print(log, x)
--> =ac	2

log = ""
x = v("a", 1) and v("b", nil) or v("c", 2) and v("d", 3)
print(log, x)
--> =abcd	3

log = ""
x = (v("a", nil) or v("b", 1)) and v("c", false) or v("d", 4)
print(log, x)
--> =abcd	4

log = ""
x = v("a", 1) or v("b", 2) and v("c", 3)
print(log, x)
--> =a	1
