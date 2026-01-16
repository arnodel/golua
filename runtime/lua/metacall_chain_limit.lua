-- Test __call metamethod chain limit (Lua 5.5)
-- A chain of __call metamethods can have at most 15 objects

local function make_chain(depth)
    if depth == 0 then
        return function() return "success" end
    end
    return setmetatable({}, {__call = make_chain(depth - 1)})
end

-- Test chains within limit
do
    print(make_chain(1)())
    --> =success
end

do
    print(make_chain(5)())
    --> =success
end

do
    print(make_chain(10)())
    --> =success
end

do
    print(make_chain(14)())
    --> =success
end

do
    print(make_chain(15)())
    --> =success
end

-- Test chains exceeding limit
do
    print(pcall(make_chain(16)))
    --> ~^false\t.*'__call' chain too long
end

do
    print(pcall(make_chain(20)))
    --> ~^false\t.*'__call' chain too long
end

do
    print(pcall(make_chain(100)))
    --> ~^false\t.*'__call' chain too long
end

-- Test that normal recursive calls still work
do
    local function factorial(n)
        if n <= 1 then return 1 end
        return n * factorial(n - 1)
    end
    print(factorial(10))
    --> =3628800
end

-- Test mixed scenario: chain with 3 objects
do
    local obj3 = function() return "reached function" end
    local obj2 = setmetatable({}, {__call = obj3})
    local obj1 = setmetatable({}, {__call = obj2})
    print(obj1())
    --> =reached function
end
