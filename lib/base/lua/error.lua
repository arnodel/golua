do
    local function test(...)
        print(pcall(error, ...))
    end
    test("hello")
    --> ~false.*: hello

    test(true)
    --> =false	true

    test("hi", 0)
    --> ~false\t.*hi

    test()
    --> ~false\t.*<no error object>

    test("foo", "bar")
    --> ~false\t.*must be an integer

    test("baz", 2)
    --> ~false.*: baz
end

-- Lua 5.5: nil error object tests
do
    print(pcall(error, nil))
    --> ~false\t.*<no error object>
end

do
    print(pcall(error, nil, 0))
    --> ~false\t.*<no error object>
end

do
    print(pcall(error, nil, 1))
    --> ~false\t.*<no error object>
end

do
    print(pcall(error, nil, 2))
    --> ~false\t.*<no error object>
end

-- Compare: string errors get file:line context, nil errors don't
do
    local ok1, err1 = pcall(function() error("string error") end)
    local ok2, err2 = pcall(function() error(nil) end)
    -- err1 should have file:line prefix like "...error.lua:47: string error"
    -- err2 should just be "<no error object>" without any prefix
    print(err1:match(":%d+: string error") ~= nil)
    --> =true
    print(err2:match(":%d+:") == nil)
    --> =true
    print(err2 == "<no error object>")
    --> =true
end

-- Critical: nil errors with explicit level should still be treated as level 0
do
    local function helper()
        error(nil, 2)  -- level 2 should be ignored, treated as level 0
    end
    local ok, err = pcall(function() helper() end)
    -- Should NOT have file:line prefix despite level 2
    print(err == "<no error object>")
    --> =true
end

do
    local ok, err = pcall(function() error(nil, 1) end)
    -- level 1 should be ignored too
    print(err == "<no error object>")
    --> =true
end
