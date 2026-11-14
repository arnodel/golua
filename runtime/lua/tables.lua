local t = {3, 2, 1, x = "abc", ["zz"] = 12}
print(t[1])
--> =3
print(#t)
--> =3
t[4]=2
print(#t)
--> =4
print(t["x"] .. t.zz)
--> =abc12

t[6]=1
t[5]=1
print(#t)
--> =6

t[6]=nil
print(#t)
--> =5

t[4]=nil
t[5]=nil
print(#t)
--> =3

t[3.2] = 5
print(t[3.2])
--> =5

t[5e2] = "hi"
print(t[500])
--> =hi

print(#t)
--> =3

t.xxx = nil
print(t.xxx)
--> =nil

print(pcall(function() t[nil] = 2 end))
--> ~false\t.*index is nil

print(pcall(function() t[0/0] = 2 end))
--> ~false\t.*index is NaN

do
    local t = {"x", "y"}
    local a, x = next(t)
    local b, y = next(t, a)
    if a < b then
        print(a..b, x..y)
    else
        print(b..a, y..x)
    end
    --> =12	xy

    print(next(t, b))
    --> =nil

    print(pcall(next, t, "abc"))
    --> ~^false

    t[b] = nil
    print(next(t, a))
    --> =nil

    print(next({}))
    --> =nil
end

-- custom length as used in table module functions
do
    local t = {"x", "y"}
    debug.setmetatable(t, {__len=function() return 10 end})
    table.insert(t, 5)
    print(t[11])
    --> =5
end
do
    local t = {"x", "y"}
    debug.setmetatable(t, {__len=function() return "hi" end})
    print(pcall(table.insert, t, 5))
    --> ~false\t.*
end
do
    local t = {"x", "y"}
    debug.setmetatable(t, {__len=function() error("haha") end})
    print(pcall(table.insert, t, 5))
    --> ~false\t.* haha
end

do
    local t = {[0]=0, [1]=1, [2]=2}
    local s = ""
    for _, x in pairs(t) do
        s = s..x
    end
    print(s)
    --> =120
end

-- Appending grows the array part, which takes over the integer keys it
-- grows past.
do
    local t = {}
    for i = 1, 4 do t[i] = i end
    t[10] = "ten"
    for i = 5, 12 do
        if i ~= 10 then t[i] = i end
    end
    print(#t, t[10], t[12], t[13])
    --> =12	ten	12	nil
    local n, sum = 0, 0
    for k, v in pairs(t) do
        n = n + 1
        if k ~= 10 then sum = sum + v end
    end
    print(n, sum)
    --> =12	68

    -- A queue, written at one end and emptied at the other.
    local q, head = {}, 1
    for tail = 1, 1000 do
        q[tail] = tail
        if tail - head >= 3 then
            q[head] = nil
            head = head + 1
        end
    end
    print(head, q[head], q[1000], q[head - 1])
    --> =998	998	1000	nil

    -- Appending to a table that has a metatable: __newindex is asked for a
    -- key that has no value, even one the array part has room for.
    local log = {}
    local p = setmetatable({}, {__newindex = function(t, k, v) log[#log + 1] = k; rawset(t, k, v) end})
    for i = 1, 6 do p[i] = i * 10 end
    p[3] = 33
    print(#p, p[3], #log, log[6])
    --> =6	33	6	6
end
