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
