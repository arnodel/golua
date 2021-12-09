local function test(...)
    local ok, val = pcall(setmetatable, ...)
    if ok then
        print("ok")
    else
        print(val)
    end
end

test()
--> ~.*: 2 arguments needed

test(1)
--> ~.*: 2 arguments needed

test(1, 2)
--> ~.*must be a table

t = {}
setmetatable(t, {__metatable={}})
test(t, nil)
--> ~.*: cannot set metatable

t = {}
test(t, nil)
--> =ok

test(t, 123)
--> ~.*must be a table
