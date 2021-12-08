do
    print(#"abc")
    --> =3

    local t = {1, 2}

    print(#t)
    --> =2

    local f = function() return 3 end
    setmetatable(t, {__len=function() return f() end})

    print(#t)
    --> =3

    f = function() error("hi", 0) end
    print(pcall(function() return #t end))
    --> =false	hi

    print(pcall(function() return #1 end))
    --> ~^false\t
end

do
    print(type"abc", type{}, type(true), type(1), type(function() end))
    --> =string	table	boolean	number	function
    
    local t = coroutine.create(function() end)
    print(type(t))
    --> =thread

    print(type(nil))
    --> =nil

    print(type(io.stdout))
    --> =file
end
