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

-- Each iteration has its own loop variable, which a closure can capture.
do
    local fs = {}
    for i = 1, 3 do fs[i] = function() return i end end
    print(fs[1](), fs[2](), fs[3]())
    --> =1	2	3
end

-- So has each iteration of a generic for loop, for every one of its
-- variables: a closure called after the loop has ended, when the iterator's
-- last call has returned nil, still sees the value of its own iteration.
do
    local fs = {}
    for i, v in ipairs{"a", "b", "c"} do fs[i] = function() return i, v end end
    print(fs[1]())
    --> =1	a
    print(fs[3]())
    --> =3	c

    local function inside()
        local gs = {}
        for k, v in pairs{x = 1} do gs[1] = function() return k, v end end
        return gs[1]()
    end
    print(inside())
    --> =x	1

    -- Called while the loop runs, and leaving it with break.
    local seen, last = {}
    for i, v in ipairs{5, 6, 7} do
        local f = function() return v end
        seen[i] = f()
        if i == 2 then last = f break end
    end
    print(seen[1], seen[2], seen[3], last())
    --> =5	6	nil	6

    -- Nested loops: each closure has the variables of both its iterations.
    local hs = {}
    for i, a in ipairs{"p", "q"} do
        for j, b in ipairs{"r", "s"} do
            hs[#hs + 1] = function() return a .. b end
        end
    end
    print(hs[1](), hs[2](), hs[3](), hs[4]())
    --> =pr	ps	qr	qs
end

-- And so has each execution of a local statement that takes its values from
-- a call: a closure that changes such a local changes its own.
do
    local function two() return 1, 2 end
    local fs = {}
    for i = 1, 2 do
        local a, b = two()
        fs[i] = function() b = b + 1 return a, b end
    end
    print(fs[1]())
    --> =1	3
    print(fs[2]())
    --> =1	3
    print(fs[1]())
    --> =1	4

    local n = 0
    local function next3() n = n + 1 if n <= 3 then return n, n * 10 end end
    local gs = {}
    while true do
        local k, v = next3()
        if not k then break end
        gs[k] = function() return v end
    end
    print(gs[1](), gs[2](), gs[3]())
    --> =10	20	30
end

-- The body reads the loop variable as the loop counts it, also after a branch.
do
    local t = {}
    for i = 3, 1, -1 do
        if i % 2 == 1 then t[#t+1] = i else t[#t+1] = -i end
    end
    print(table.concat(t, ","))
    --> =3,-2,1
end
