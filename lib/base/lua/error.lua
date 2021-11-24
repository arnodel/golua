do
    local function test(...)
        print(pcall(error, ...))
    end
    test("hello")
    --> =false	hello

    test(true)
    --> =false	true

    test("hi", 0)
    --> ~false\t.*must be > 0

    test()
    --> ~false\t.*value needed

    test("foo", "bar")
    --> ~false\t.*must be an integer

    test("baz", 2)
    --> =false	baz
end
