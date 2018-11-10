do
    local function test(...)
        print(pcall(getmetatable, ...))
    end

    test(1)
    --> =true	nil

    test({})
    --> =true	nil

    test()
    --> ~false\t.*value needed
end
