local function countsteps(start, stop, step)
    local n = 0
    if step then
        for i = start, stop, step do
            n = n + 1
        end
    else
        for i = start, stop do
            n = n + 1
        end
    end
    return n
end

print(countsteps(1, 3))
--> =3

print(countsteps(math.maxinteger - 1, 1e100))
--> =2

print(countsteps(1, -10))
--> =0

print(countsteps(math.mininteger, math.mininteger + 1))
--> =2

print(countsteps(math.mininteger + 2, -1e100, -1))
--> =3

print(countsteps(1.0, 6, 2))
--> =3

for i = 1, 2, 1.1 do
    print(math.type(i))
end
--> =float

print(pcall(function() for i = 1, 1, 0 do end end))
--> ~false\t.*'for' step is zero

-- Errors
do
    local function err(init)
        local f = load('for i = ' .. init .. ' do end')
        ok, msg = pcall(f)
        print(msg)
    end

    err[['a', 2, 3]]
    --> ~'for' initial value: expected number, got string

    err[[1, {}]]
    --> ~'for' limit: expected number, got table

    err[[1, 2, false]]
    --> ~'for' step: expected number, got boolean

    err[[1, 2, 0]]
    --> ~'for' step is zero
end

-- An integer loop with a float limit rounds the limit towards the start, and
-- its variable stays an integer.
do
    local t = {}
    for i = 1, 2.5 do t[#t+1] = i end
    print(table.concat(t, ","))
    --> =1,2
    t = {}
    for i = 3, 0.5, -1 do t[#t+1] = i end
    print(table.concat(t, ","))
    --> =3,2,1
    for i = 1, 1.5 do print(math.type(i)) end
    --> =integer
end

-- A limit beyond the integer range is clipped to it, and the loop stops at
-- overflow rather than wrapping around.
do
    local n = 0
    for i = math.maxinteger - 2, 1e100 do n = n + 1 end
    print(n)
    --> =3
    n = 0
    for i = math.mininteger + 2, -1e100, -1 do n = n + 1 end
    print(n)
    --> =3
end

-- A limit on the wrong side of the integer range means no iteration, and
-- neither does a NaN limit with a positive step.  With a negative step a NaN
-- limit counts as below the range, as in Lua 5.4.
do
    local n = 0
    for i = 1, -1e100 do n = n + 1 end
    for i = -1, 1e100, -1 do n = n + 1 end
    for i = 1, 0/0 do n = n + 1 end
    print(n)
    --> =0
    for i = 1, 0/0, -1 do n = n + 1 if n == 6 then break end end
    print(n)
    --> =6
end

-- A float start or step makes a float loop whatever the limit.
do
    local t = {}
    for i = 1.0, 3 do t[#t+1] = i end
    print(table.concat(t, ","))
    --> =1.0,2.0,3.0
    t = {}
    for i = 1, 2, 0.5 do t[#t+1] = i end
    print(table.concat(t, ","))
    --> =1.0,1.5,2.0
end
