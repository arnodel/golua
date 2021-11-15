quota = require'quota'

--
-- Check that recursion consumes memory
--

-- Naive recursive fibonacci implementation
function rfib(n)
    if n < 2 then return n
    else return rfib(n-1) + rfib(n-2)
    end
end

-- Check the implementation is correct
print(rfib(1), rfib(2), rfib(3), rfib(10))
--> =1	1	2	55

-- Recursion blows memory budget
print(quota.rcall(0, 1000, rfib, 100))
--> =false

-- Recursion blows cpu budget
print(quota.rcall(10000, 0, rfib, 100))
--> =false

--
-- Check that iteration consumes CPU
--

-- Iterative fibonacci implementation
function ifib(n)
    local a, b = 0, 1
    while n > 0 do
        a, b = b, a+b
        n = n - 1
    end
    return a
end

-- Check the implementation is correct
print(ifib(1), ifib(2), ifib(3), ifib(10))
--> =1	1	2	55

-- memory usage doesn't explode
print(quota.rcall(0, 1000, ifib, 100))
--> =true	3736710778780434371

-- cpu usage doesn't explode
print(quota.rcall(10000, 0, ifib, 100))
--> =true	3736710778780434371

-- we can run out of cpu eventually!
print(quota.rcall(1000, 0, ifib, 1000))
--> =false

--
-- Check that tail recursion does not consume memory
--

-- Tail-recursive fibonacci implementation
function trfib(n, a, b)
    a, b = a or 0, b or 1
    if n == 0 then return a end
    return trfib(n - 1, b, a + b)
end

-- Check the implementation is correct
print(trfib(1), trfib(2), trfib(3), trfib(10))
--> =1	1	2	55

-- memory usage doesn't explode
print(quota.rcall(0, 1000, ifib, 100))
--> =true	3736710778780434371

-- cpu usage doesn't explode
print(quota.rcall(10000, 0, ifib, 100))
--> =true	3736710778780434371

-- we can run out of cpu eventually!
print(quota.rcall(1000, 0, ifib, 1000))
--> =false

--
-- Check that strinc concatenation consumes memory
--

-- strpex(x, n) is x concatenated to itself 2^n timees
function strexp(s, n)
    if n == 0 then
        return ""
    end
    while n > 1 do
        s = s..s
        n = n - 1
    end
    return s
end

print(strexp("hi", 2))
--> =hihi

print(strexp("hi", 3))
--> =hihihihi

--> strexp doesn't consume much cpu
ok, bigs = quota.rcall(1000, 0, strexp, "hi", 16)
print(ok, #bigs)
--> =true	65536

--> but it consumes memory!
print(quota.rcall(0, 50000, strexp, "hi", 16))
--> =false

--
-- Check that it costs memory to pass many arguments to a function
--

table = require'table'

function len(...)
    return select('#', ...)
end

function numbers(n)
    local t = {}
    for i = 1, n do
        t[i] = i
    end
    return t
end

print(table.unpack(numbers(5)))
--> =1	2	3	4	5

print(len(table.unpack(numbers(4))))
--> =4

print(quota.rcall(0, 1000, len, table.unpack(numbers(10))))
--> =true	10

-- Passing a long list of arguments requires a lot of memory.
print(quota.rcall(0, 8000, len, table.unpack(numbers(500))))
--> =false
