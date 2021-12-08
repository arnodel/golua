do
    local function test(...)
        print(pcall(error, ...))
    end
    test("hello")
    --> =false	hello

    test(true)
    --> =false	true

    test("hi", 0)
    --> ~false\t.*hi

    test()
    --> ~false\t.*nil

    test("foo", "bar")
    --> ~false\t.*must be an integer

    test("baz", 2)
    --> =false	baz
end
