local function counter(start, step)
    return function()
        local val = start
        start = start + step
        return val
    end
end

local nxt = counter(5, 3)
print(nxt(), nxt(), nxt(), nxt())
--> =5	8	11	14

local function fib(a, b)
    return function()
        local c = a
        a, b = b, a + b
        return c
    end
end

local f = fib(1, 1)
print(f(), f(), f(), f(), f(), f())
--> =1	1	2	3	5	8
