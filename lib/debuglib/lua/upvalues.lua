local function perr(...)
    local ok, err = pcall(...)
    if not ok then
        print(err)
    end
end

-- getupvalue tests
do
    local function outer(x, y, z)
        return function()
            return x, y, z
        end
    end

    local inner = outer("hello", 1, false)

    for i = 1, 4 do
        print(debug.getupvalue(inner, i))
    end

    --> =x	hello
    --> =y	1
    --> =z	false
    --> =

    function perr(...)
        local ok, err = pcall(...)
        if not ok then
            print(err)
        end
    end

    perr(debug.getupvalue, inner)
    --> =2 arguments needed

    perr(debug.getupvalue, 1, 1)
    --> =#1 must be a lua function

    perr(debug.getupvalue, inner, "a")
    --> =#2 must be an integer
end

-- setupvalue tests
do
    local function outer(x, y, z)
        return function()
            return x, y, z
        end
    end

    local inner = outer("hello", 1, false)

    print(debug.setupvalue(inner, 1, "bye"))
    --> =x
    print(debug.setupvalue(inner, 2, 42))
    --> =y
    print(debug.setupvalue(inner, 3, true))
    --> =z
    print(debug.setupvalue(inner, 4, "non-existent"))
    --> =

    print(inner())
    --> =bye	42	true

    perr(debug.setupvalue, inner, 1)
    --> =3 arguments needed

    perr(debug.setupvalue, {}, 1, 2)
    --> =#1 must be a lua function

    perr(debug.setupvalue, inner, "x", 3)
    --> =#2 must be an integer
end
