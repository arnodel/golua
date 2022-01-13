--
-- const tests
--

local function test(src)
    local res, err = load(src)
    if res ~= nil then
        print(pcall(res))
    else
        print(false, err)
    end
end

test[[
    local foo <const> = 25
    return foo
]]
--> =true	25

test[[
    local foo <const> = "hello"
    foo = "bye"
]]
--> ~false\t.*attempt to reassign constant variable 'foo'

test[[
    local foo <const> = 25
    local foo = foo + 2
    return foo
]]
--> =true	27

test[[
    local foo, bar <const> = 25, 32
    do
        local bar
        bar = 72
    end
    foo = foo + 2
    bar = bar - 1
    return foo + bar
]]
--> ~false\t.*attempt to reassign constant variable 'bar'

--
-- to-be-closed tests
--

test[[
    local x <close> = nil
    x = 3
]]
--> ~false\t.*attempt to reassign constant variable 'x'

test[[
    local x <close> = 1
]]
--> ~false\t.*to be closed variable missing a __close metamethod

function make(msg, err)
    t = {}
    setmetatable(t, {__close = function (x, e) 
        if e ~= nil then
            print(msg, e)
        else
            print(msg)
        end
        if err ~= nil then 
            error(err)
        end
    end})
    return t
end

do
    local x <close> = make("x")
    print("a")
end
print("b")
--> =a
--> =x
--> =b

do
    local x <close> = make("x")
    local y <close> = make("y")
end
--> =y
--> =x

do
    local x <close>, y <close> = make("aa"), make("bb")
end
--> =bb
--> =aa

-- errors in close metamethods.  If a one produces an error, it looks like the
-- next one is fed that error.
pcall(function()
    local x <close> = make("x")
    local y <close> = make("y", "YY")
    local z <close> = make("z")
    local t <close> = make("t", "TT")
    error("ERROR")
end)
--> ~t\t.*ERROR
--> ~z\t.*TT
--> ~y\t.*TT
--> ~x\t.*YY

-- A function to test to-be-closed variables
local s
function mk(a)
    t = {}
    s = s .. '+' .. a
    setmetatable(t, {__close = function () s = s .. '-' .. a end})
    return t
end

-- How it works
do
    s = "start"
    local v <close> = mk("bob")
    print(s)
    --> =start+bob
end
print(s)
--> =start+bob-bob

do
    s = "start"
    local function f()
        local a <close> = mk("a")
        for i = 1, 3 do
            local b <close> = mk("b"..i)
        end
        do
            local c <close> = mk("c")
            do
                local d <close> = mk("d")
                return
            end
        end
    end
    f()
    print(s)
    --> =start+a+b1-b1+b2-b2+b3-b3+c+d-d-c-a
end

do
    s = "start"
    local function f(n)
        local x <close> = mk("x"..n)
        if n > 0 then
            f(n - 1)
        else
            error("stop")
        end
    end
    print(pcall(f, 3))
    --> ~false\t.*: stop
    print(s)
    --> =start+x3+x2+x1+x0-x0-x1-x2-x3
end

-- Tail calls are disabled when there are pending to-be-closed variables.
do
    s = "start"
    local function g()
        local y <close> = mk("y")
    end
    local function f()
        local x <close> = mk("x")
        return g() -- This isn't a tail call
    end
    f()
    print(s)
    --> =start+x+y-y-x
end
