do
    local function f(...)
        print(...)
    end
    f(1, 2)
    --> =1	2

    local function f(...)
        local a, b, c = ..., ...
        print(a, b, c)
    end
    f(1, 2, 3)
    --> =1	1	2

    f(1)
    --> =1	1	nil

    f()
    --> =nil	nil	nil
end
