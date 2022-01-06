
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
