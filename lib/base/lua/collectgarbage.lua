do
    local function test(...)
        print(pcall(collectgarbage, ...))
    end

    test()
    --> =true

    test("collect")
    --> =true

    test("step")
    --> =true	true

    test("isrunning")
    --> =true	true

    test("stop")
    --> =true

    test("isrunning")
    --> =true	false

    test("restart")
    --> =true

    test("isrunning")
    --> =true	true

    test("setpause")
    --> =true

    test("setstepmul")
    --> =true

    test("count")
    --> ~true\t

    test("blah")
    --> ~false\t.*: invalid option

    test(1)
    --> ~false\t.*must be a string
end