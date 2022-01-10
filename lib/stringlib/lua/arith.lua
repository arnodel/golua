local n = {}
local meta = {}
setmetatable(n, meta)
meta.__add = function(x, y)
    if x == "BOOM" then
        error(x)
    end
    return "<add>"
end

do
    print("1" + "2", 1 + "3", "2.4" + 2)
    --> =3	4	4.4

    print(pcall(function() return "a" + "1" end))
    --> ~false\t.*attempt to perform arithmetic on a string value

    print(pcall(getmetatable("12").__add))
    --> ~false\t.*2 arguments needed

    print(n + 1, 1 + n, n + "a", "a" + n)
    --> =<add>	<add>	<add>	<add>

    print(pcall(function() return "BOOM" + n end))
    --> ~false\t.*BOOM
end
