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

do
    local function c(t)
        print(table.concat(t, '-'))
    end

    local function f(...)
        c({...})
        c({"a", "b", ...})
    end
    f(1, 2)
    --> =1-2
    --> =a-b-1-2

    local function g()
        return 3, 2, 1
    end
    c({g()})
    --> =3-2-1

    c({1, 2, g()})
    --> =1-2-3-2-1

    c({g(), g()})
    --> =3-3-2-1
end

do
    local function f()
        return 1, 2
    end

    print(f())
    --> =1	2

    print((f()))
    --> =1

    local a, b = (f())
    print(b)
    --> =nil

    local t = {f()}
    print(#t)
    --> =2

    t = {(f())}
    print(#t)
    --> =1
end
