do
    print(pcall(table.concat))
    --> ~^false\t.*value needed

    print(pcall(table.concat, "not a table"))
    --> ~^false\t.*must be a table

    print(pcall(table.concat, {}, 1))
    --> ~^false\t.*must be a string

    print(pcall(table.concat, {}, "--", false))
    --> ~^false\t.*must be an integer

    print(pcall(table.concat, {}, "--", 1, {}))
    --> ~^false\t.*must be an integer

    local t = {1, 2, 3}

    print(table.concat(t))
    --> =123

    print(table.concat(t, "--"))
    --> =1--2--3

    print(table.concat({}))
    --> =

    print(type(table.concat({})))
    --> =string

    print(table.concat({"foo"}))
    --> =foo

    print(table.concat(t, "", 2, 3))
    --> =23

    print(table.concat(t, "", 2, 2))
    --> =2

    print(table.concat(t, "", 3, 2))
    --> =

    t[-1]="hel"
    t[0]="lo"
    print(table.concat(t, "", -1, 1))
    --> =hello1

    print(pcall(table.concat(t, "", 2, 5)))
    --> ~^false\t.*
end

do
    print(pcall(table.insert))
    --> ~^false\t.*2 arguments needed

    print(pcall(table.insert, 1, false))
    --> ~^false\t.*must be a table

    print(pcall(table.insert, {}, "hello", true))
    --> ~^false\t.*must be an integer

    local t = {1, 2, 3}
    table.insert(t, "foo")
    print(t[4])
    --> =foo

    table.insert(t, 2, 42)
    print(t[2], t[3], #t)
    --> =42	2	5

    print(pcall(table.insert, t, -1, 1))
    --> ~^false\t.*

    local tt = {}
    setmetatable(tt, {
        __len=function() return 3 end,
        __index=function() error("g") end
    })
    print(pcall(table.insert, tt, 1, 12))
    --> =false	g

    local tt = {}
    setmetatable(tt, {
        __len=function() return 3 end,
        __index=function() return 2 end,
        __newindex=function() error("s") end
    })
    print(pcall(table.insert, tt, 1, 12))
    --> =false	s

    print(pcall(table.insert, tt, 4, 123))
    --> =false	s
end

do
    print(pcall(table.move))
    --> ~^false\t.*4 arguments needed

    print(pcall(table.move, 1, 2, 3, 4))
    --> ~^false\t.*must be a table

    print(pcall(table.move, {}, true, 3, 4))
    --> ~^false\t.*must be an integer

    print(pcall(table.move, {}, 2, false, 4))
    --> ~^false\t.*must be an integer

    print(pcall(table.move, {}, 2, 3, "xxx"))
    --> ~^false\t.*must be an integer

    print(pcall(table.move, {}, 1, 2, 3, "bar"))
    --> ~^false\t.*must be a table

    local t = {1, 2, 3, 4}
    table.move(t, 2, 4, 3)
    print(table.concat(t))
    --> =12234

    table.move(t, 3, 5, 2)
    print(table.concat(t))
    --> =12344

    local u = {}
    print(table.concat(table.move(t, 1, 4, 1, u)))
    --> =1234
end

do
    local t = table.pack(3, 2, 1, 4, 5)
    print(t.n, #t)
    --> =5	5
    print(table.concat(t))
    --> =32145
end

do
    print(pcall(table.remove))
    --> ~^false\t.*value needed

    print(pcall(table.remove, 1))
    --> ~^false\t.*must be a table

    print(pcall(table.remove, {}, true))
    --> ~^false\t.*must be an integer

    print(pcall(table.remove, {}, 2))
    --> ~^false\t.*out of range

    local t = {1, 2, 3, 4, 5}
    print(table.remove(t))
    --> =5

    print(table.remove(t))
    --> =4

    print(table.remove(t, 2))
    --> =2

    print(table.concat(t))
    --> =13

    print(table.remove({}))
    --> =nil

    print(table.remove({}, 0))
    --> =nil

    print(table.remove({}, 1))
    --> =nil

    print(table.remove(t, 3))
    --> =nil
          
    local tt = {}
    setmetatable(tt, {
        __len=function() return 3 end,
        __index=function() error("g") end
    })
    print(pcall(table.remove, tt))
    --> =false	g

    local tt = {}
    setmetatable(tt, {
        __len=function() return 3 end,
        __index=function(n, i) return -i end,
        __newindex=function() error("s") end
    })
    print(pcall(table.remove, tt))
    --> =false	s

    print(pcall(table.remove, tt, 2))
    --> =false	s
end

do
    print(pcall(table.sort))
    --> ~^false\t.*value needed

    print(pcall(table.sort, 1))
    --> ~^false\t.*must be a table

    local t = {3, 2, 4, 1, 5}
    table.sort(t)
    print(table.concat(t))
    --> =12345

    table.sort(t, function(x, y) return x > y end)
    print(table.concat(t))
    --> =54321

    local t = {"bar", "bat", "fur", "ball", "four"}
    table.sort(t)
    print(table.concat(t, " "))
    --> =ball bar bat four fur

    local tt = {}
    setmetatable(tt, {
        __len=function() return 3 end,
        __index=function() error("g") end
    })
    print(pcall(table.sort, tt))
    --> =false	g

    local tt = {}
    setmetatable(tt, {
        __len=function() return 3 end,
        __index=function(n, i) return -i end,
        __newindex=function() error("s") end
    })
    print(pcall(table.sort, tt))
    --> =false	s
end

do
    print(pcall(table.unpack))
    --> ~^false\t.*value needed

    print(pcall(table.unpack, 1))
    --> ~^false\t.*must be a table

    print(pcall(table.unpack, {}, "a"))
    --> ~^false\t.*must be an integer

    print(pcall(table.unpack, {}, 2, "a"))
    --> ~^false\t.*must be an integer

    print(table.unpack({3, 4, 1, 5}))
    --> =3	4	1	5

    print(table.unpack({1, 2, 3, 4, 5, 6}, 3, 5))
    --> =3	4	5

    print(table.unpack({3, 2, 1}, 3, 5))
    --> =1	nil	nil

    print(table.unpack({4, 3, 2}, -1, 1))
    --> =nil	nil	4

    local tt = {}
    setmetatable(tt, {
        __len=function() return 3 end,
        __index=function() error("g") end
    })
    print(pcall(table.unpack, tt))
    --> =false	g
end
