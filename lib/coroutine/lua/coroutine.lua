-- Some general checks
do
    local function cof(x)
        print("in cof", x)
        print("in cof", coroutine.yield(x + 2))
        return "from cof"
    end

    local co = coroutine.create(cof)
    print("out", coroutine.resume(co, 1))
    print("out", coroutine.resume(co, "two"))
    
    --> =in cof	1
    --> =out	true	3
    --> =in cof	two
    --> =out	true	from cof
end

-- Check the main coroutine is correctly identified
print(coroutine.running())
--> ~^thread:.*\ttrue$

-- Check the main coroutine is not yieldable
print(coroutine.isyieldable())
--> =false

-- Check that coroutine.running() returns true as second argument when
-- called from a non-main coroutine and that a non main coroutine is
-- yieldable.
do
    local function cof()
        print("yieldable", coroutine.isyieldable())
        return coroutine.running()
    end
    local co = coroutine.create(cof)

    print(coroutine.resume(co))
    --> =yieldable	true
    --> ~^true\tthread:.*\tfalse$
end

-- Test error in coroutine
do
    local function cof()
        error("boo")
    end
    local co = coroutine.create(cof)
    print(coroutine.resume(co))
    --> ~^false\tboo$
    print(coroutine.status(co))
    --> =dead
end

-- Check various statuses of coroutines
do
    local main = coroutine.running()
    local function cof(co)
        coroutine.yield(coroutine.status(co))
        return coroutine.status(main)
    end
    co = coroutine.create(cof)

    print("main/main", coroutine.status(main))
    --> =main/main	running

    print("co/main", coroutine.status(co))
    --> =co/main	suspended

    print("co/co", coroutine.resume(co, co))
    --> =co/co	true	running

    print("main/co", coroutine.resume(co))
    --> =main/co	true	normal

    print("co", coroutine.status(co))
    --> =co	dead
end

-- Test coroutine.wrap()
do
    local function cofib()
        local a, b = 0, 1
        while true do
            coroutine.yield(a)
            a, b = b, a+b
        end
    end
    local fib = coroutine.wrap(cofib)
    print(fib(), fib(), fib(), fib(), fib())
    --> =0	1	1	2	3
end
