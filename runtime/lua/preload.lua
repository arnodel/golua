-- Constant operands are read from registers set when each call starts.

-- Every kind of constant, on either side of an operation, as a table key and
-- as a stored value.
local function ops(x, t)
    t[1] = x + 1
    t.f = x * 2.5
    t[3] = 10 - x
    t[4] = x == nil
    t[5] = x ~= true
    t[6] = "<" .. tostring(x) .. ">"
    t[7] = x < 100
    t["k"] = false
    return t[1], t.f, t[3], t[4], t[5], t[6], t[7], t.k
end

print(ops(4, {}))
--> =5	10.0	6	false	true	<4>	true	false

-- The same in a function loaded from a binary chunk.
local dumped = load(string.dump(ops))
print(dumped(4, {}))
--> =5	10.0	6	false	true	<4>	true	false

-- Each call has its own registers: recursion and coroutines see the
-- constants intact.
local function depth(t)
    if t[1] == nil then return 1 end
    return depth(t[1]) + depth(t[2]) + 1
end
print(depth({{{}, {}}, {}}))
--> =5

local co = coroutine.wrap(function(n)
    while true do n = coroutine.yield(n + 1) * 2 end
end)
print(co(1), co(10), co(20))
--> =2	21	41

-- A function with too many registers for preloaded constants is compiled
-- without them.  (C Lua allows no more than 200 locals.)
local names = {}
for i = 1, 252 do names[i] = "a" .. i end
local many = load("local " .. table.concat(names, ", ") .. " = ...\n" ..
    "return a1 + 1, a2 - 2.5, a3 == 'x', a4 == nil")
print(many(1, 2, "x", 4))
--> =2	-0.5	true	false
