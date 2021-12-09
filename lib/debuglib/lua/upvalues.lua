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
    --> ~.*: 2 arguments needed

    perr(debug.getupvalue, 1, 1)
    --> ~.*: #1 must be a lua function

    perr(debug.getupvalue, inner, "a")
    --> ~.*: #2 must be an integer
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
    --> ~.*: 3 arguments needed

    perr(debug.setupvalue, {}, 1, 2)
    --> ~.*: #1 must be a lua function

    perr(debug.setupvalue, inner, "x", 3)
    --> ~.*: #2 must be an integer
end

-- upvaluejoin tests
do
    local function maker(...)
        local function outer(x, y, z)
            return function()
                return x, y, z
            end
        end
        return outer(...)
    end

    local f1 = maker(1, 2, 3)
    local f2 = maker("a", "b", "c")

    debug.upvaluejoin(f1, 1, f2, 2)
    debug.upvaluejoin(f2, 1, f1, 2)

    print(f1())
    --> =b	2	3

    print(f2())
    --> =2	b	c

    debug.setupvalue(f1, 1, "x")
    debug.setupvalue(f1, 2, "x")
    debug.setupvalue(f1, 3, "x")

    print(f1())
    --> =x	x	x

    print(f2())
    --> =x	x	c

    perr(debug.upvaluejoin, f1, 1, f2)
    --> ~.*: 4 arguments needed

    perr(debug.upvaluejoin, "x", 1, f2, 2)
    --> ~.*: #1 must be a lua function

    perr(debug.upvaluejoin, f1, "x", f2, 2)
    --> ~.*: #2 must be an integer

    perr(debug.upvaluejoin, f1, 1, "x", 2)
    --> ~.*: #3 must be a lua function

    perr(debug.upvaluejoin, f1, 1, f2, "x")
    --> ~.*: #4 must be an integer

    perr(debug.upvaluejoin, f1, 1, f2, 4)
    --> ~.*: Invalid upvalue index
end

-- upvalueid teests
do
    local f1, f2
    local function outer(x, y, z)
        local a, b = 1, 2
        f1 = function ()
            return x, y, a
        end
        f2 = function()
            return a, b, x
        end
    end
    outer()

    local id = debug.upvalueid

    print(id(f1, 1) == id(f2, 3))
    --> =true

    print(id(f1, 2) == id(f2, 3))
    --> =false

    print(id(f1, 3) == id(f2, 1))
    --> =true

    perr(id, f1)
    --> ~.*: 2 arguments needed

    perr(id, print, 2)
    --> ~.*: #1 must be a lua function

    perr(id, f1, nil)
    --> ~.*: #2 must be an integer

    perr(id, f1, 0)
    --> ~.*: Invalid upvalue index

    perr(id, f1, 4)
    --> ~.*: Invalid upvalue index
end
