
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

test[[
    local x <close> = nil
    x = 3
]]
--> ~false\t.*attempt to reassign constant variable 'x'

test[[
    local x <close> = 1
]]
--> ~false\t.*to be closed variable missing a __close metamethod

function make(msg)
    t = {}
    setmetatable(t, {__close = function () print(msg) end})
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

local s = "start"
function mk(a)
    t = {}
    s = s .. '+' .. a
    setmetatable(t, {__close = function () s = s .. '-' .. a end})
    return t
end

do
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
