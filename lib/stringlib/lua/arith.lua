local n = {}
local meta = {}
setmetatable(n, meta)
meta.__tostring = function() return "<n>" end
meta.__add = function(x, y)
    if x == "BOOM" then
        error(x)
    end
    return x
end
meta.__idiv = function(x, y)
    if x == "BOOM" then
        error(x)
    end
    return x
end

do
    print("1" + "2", 1 + "3", "2.4" + 2)
    --> =3	4	4.4

    print(pcall(function() return "a" + "1" end))
    --> ~false\t.*attempt to perform arithmetic on a string value

    print(pcall(getmetatable("12").__add))
    --> ~false\t.*2 arguments needed

    print(n + 1, 1 + n, n + "a", "a" + n)
    --> =<n>	1	<n>	a

    print(type("1" + n))
    --> =string

    print(pcall(function() return "BOOM" + n end))
    --> ~false\t.*BOOM
end

do
    print("1" // "2", 7 // "2", "2.4" // 2)
    --> =0	3	1

    print(pcall(function() return "a" // "1" end))
    --> ~false\t.*attempt to perform arithmetic on a string value

    print(pcall(getmetatable("12").__idiv))
    --> ~false\t.*2 arguments needed

    print(n // 1, 1 // n, n // "a", "a" // n)
    --> =<n>	1	<n>	a

    print(type("1" // n))
    --> =string

    print(pcall(function() return "BOOM" // n end))
    --> ~false\t.*BOOM

    print(pcall(function() return "8" // "0" end))
    --> ~false\t.*attempt to divide by zero
end

do
    print(-"2")
    --> =-2

    print(pcall(getmetatable("12").__unm))
    --> ~false\t.*value needed

    print(pcall(function() return -"bob" end))
    --> ~false\t.*attempt to unm a 'string'
end

do
    print("2" - "2")
    --> =0

    print("3" * 5, 0.1 * "10")
    --> =15	1

    print("7" / 2, "5" / "10")
    --> =3.5	0.5

    print("3" % 2)
    --> =1

    print(2 ^ "3", "5" ^ 2, "6" ^ "3")
    --> =8	25	216
end
