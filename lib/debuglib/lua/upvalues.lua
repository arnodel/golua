
-- getupvalue tests
do
    function outer(x, y, z)
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