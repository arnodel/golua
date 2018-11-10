do
    local function test(x)
        ok, val = pcall(tostring, x)
        print(val)
    end

    test("hello")
    --> =hello
    test(123)
    --> =123
    test(nil)
    --> =nil

    t = {x=1}
    setmetatable(t, {__tostring=function(t) return t.x end})
    test(t)
    --> =1

    t.x = true
    test(t)
    --> ~.*'__tostring' must return a string

    setmetatable(t, {__tostring=1})
    test(t)
    --> ~.*expects a callable

    test({})
    --> ~table:.*

    print(pcall(tostring))
    --> ~false\t.*value needed
end
