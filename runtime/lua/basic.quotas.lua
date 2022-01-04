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
print(runtime.callcontext({memlimit=1000}, rfib, 100))
--> =killed

-- Recursion blows cpu budget
print(runtime.callcontext({cpulimit=10000}, rfib, 100))
--> =killed

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
print(runtime.callcontext({memlimit=1000}, ifib, 100))
--> =done	3736710778780434371

-- cpu usage doesn't explode
print(runtime.callcontext({cpulimit=10000}, ifib, 100))
--> =done	3736710778780434371

-- we can run out of cpu eventually!
print(runtime.callcontext({cpulimit=1000}, ifib, 1000))
--> =killed

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
print(runtime.callcontext({memlimit=1000}, ifib, 100))
--> =done	3736710778780434371

-- cpu usage doesn't explode
print(runtime.callcontext({cpulimit=10000}, ifib, 100))
--> =done	3736710778780434371

-- we can run out of cpu eventually!
print(runtime.callcontext({cpulimit=1000}, ifib, 1000))
--> =killed

--
-- Check that string concatenation consumes memory
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
ctx, bigs = runtime.callcontext({cpulimit=1000}, strexp, "hi", 16)
print(ctx, #bigs)
--> =done	65536

--> but it consumes memory!
print(runtime.callcontext({memlimit=50000}, strexp, "hi", 16))
--> =killed

--
-- Check that it costs memory to pass many arguments to a function
--

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

print(runtime.callcontext({memlimit=1000}, len, table.unpack(numbers(10))))
--> =done	10

-- Passing a long list of arguments requires a lot of memory.
print(runtime.callcontext({memlimit=2000}, len, table.unpack(numbers(200))))
--> =killed

--
-- Check soft limits and stopping
--

local c = 0
print(runtime.callcontext({stop={cpu=1000}}, function()
    print(runtime.context().stop.cpu)
    --> =1000
    while not runtime.shouldstop() do
        c = c + 1
    end
    runtime.stopcontext()
end))
--> =killed
print(c > 10)
--> =true

--
-- Check time limits
--
local c = 0
print(runtime.callcontext({kill={millis=10}}, function()
    while true do
        c = c + 1
    end
end))
--> =killed
print(c > 10)
--> =true

--
-- Check compliance flags
--
print(runtime.callcontext({flags="cpusafe"}, function()
    golib.import('fmt').Println("hello")
end))
--> ~error\t.*missing flags: cpusafe
