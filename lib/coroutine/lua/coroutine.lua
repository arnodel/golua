-- Some general checks
do
    local function cof(x)
        print("in cof", x)
        print("in cof", coroutine.yield(x + 2))
        return "from cof"
    end

    local co = coroutine.create(cof)

    print(pcall(coroutine.resume, {}))
    --> ~false\t.*#1 must be a thread

    print("out", coroutine.resume(co, 1))
    print("out", coroutine.resume(co, "two"))
    
    --> =in cof	1
    --> =out	true	3
    --> =in cof	two
    --> =out	true	from cof

    print(coroutine.status(co))
    --> =dead

    print(pcall(coroutine.status, cof))
    --> ~false\t.*#1 must be a thread

    print(pcall(coroutine.create, 1))
    --> ~false\t.*#1 must be a callable
end

-- Check the main coroutine is correctly identified
print(coroutine.running())
--> ~^thread:.*\ttrue$

do
    -- Check the main coroutine is not yieldable
    print(coroutine.isyieldable())
    --> =false

    print(pcall(coroutine.yield, 1))
    --> ~false\t.*cannot yield from main thread

    print(pcall(coroutine.isyieldable, 1))
    --> ~false\t.*must be a thread
end
 
-- Check that coroutine.running() returns true as second argument when
-- called from a non-main coroutine and that a non main coroutine is
-- yieldable.
do
    local function cof()
        print("yieldable", coroutine.isyieldable())
        return coroutine.running()
    end
    local co = coroutine.create(cof)

    print(coroutine.isyieldable(co))
    --> =true

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
    --> ~^false\t.* boo$
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

    print(pcall(coroutine.wrap))
    --> ~false\t.*value needed

    print(pcall(coroutine.wrap, "not a function"))
    --> ~false\t.*must be a callable

    local werr = coroutine.wrap(function(x, y)
        coroutine.yield(x)
        error(y)
    end)
    print(werr("abc", "def"))
    --> =abc

    print(pcall(werr))
    --> ~false\t.*def

    print(pcall(werr))
    --> ~false\t.*cannot resume dead thread
end

-- Test coroutine.close() (5.4)
do
    print(pcall(coroutine.close))
    --> ~false\t.*value needed

    print(pcall(coroutine.close, 1))
    --> ~false\t.*must be a thread

    print(pcall(coroutine.close, coroutine.running()))
    --> ~false\t.*cannot close running thread

    local co = coroutine.create(function()
        coroutine.yield()
    end)

    coroutine.resume(co)
    print(coroutine.close(co))
    --> =true
    print(coroutine.status(co))
    --> =dead
    local co = coroutine.create(function()
        local t = {}
        setmetatable(t, {__close=function() error("ERR") end})
        local x <close> = t
        coroutine.yield()
    end)
    coroutine.resume(co)
    pcall(function() print(coroutine.close(co)) end)
    --> ~false\t.*ERR

    print(coroutine.status(co))
    --> =dead
end

-- A __close handler run by the end of its coroutine cannot yield: it is
-- called by coroutine.close, or by the coroutine dying, not by Lua code that
-- could be suspended.  The yield is an error in the handler.
do
    local function closing(log)
        return setmetatable({}, {__close = function()
            log[#log + 1] = "handler"
            coroutine.yield("from the handler")
            log[#log + 1] = "not reached"
        end})
    end

    -- Closed while suspended.
    local log = {}
    local co = coroutine.create(function()
        local x <close> = closing(log)
        coroutine.yield()
    end)
    coroutine.resume(co)
    print(coroutine.close(co))
    --> ~false\t.*attempt to yield across a Go-call boundary
    print(coroutine.status(co), table.concat(log, " "))
    --> =dead	handler

    -- Dying of an error.
    log = {}
    co = coroutine.create(function()
        local x <close> = closing(log)
        error("boom")
    end)
    print(coroutine.resume(co))
    --> ~false\t.*attempt to yield across a Go-call boundary
    print(coroutine.status(co), table.concat(log, " "))
    --> =dead	handler
    print(coroutine.resume(co))
    --> ~false\t.*cannot resume dead
end

-- A coroutine is light (it has no goroutine) until it does something that
-- needs one.  Each of these makes one yield from somewhere a light coroutine
-- cannot: they must all behave the same as in a coroutine that never was.
do
    -- Yield inside pcall and xpcall.
    local co = coroutine.wrap(function()
        print(pcall(function() return coroutine.yield(1) + 1 end))
        print(xpcall(function() return coroutine.yield(2) + 1 end, print))
        return "end"
    end)
    print(co())
    --> =1
    print(co(10))
    --> =true	11
    --> =2
    print(co(20))
    --> =true	21
    --> =end

    -- Yield inside a metamethod called by an operator, then again by the
    -- same operator (the opcode runs again when the coroutine moves).
    local mt = {
        __index = function(_, k) return coroutine.yield(k) end,
        __add = function(a, b) return coroutine.yield("add") end,
    }
    co = coroutine.wrap(function()
        local x = setmetatable({}, mt)
        local a = x.first
        local b = x + 1
        local c = x.second
        return a, b, c
    end)
    print(co())
    --> =first
    print(co("A"))
    --> =add
    print(co("B"))
    --> =second
    print(co("C"))
    --> =A	B	C

    -- Yield from a __pairs metamethod, and from the iterator it returns.
    co = coroutine.wrap(function()
        local t = setmetatable({}, {__pairs = function(t)
            coroutine.yield("pairs")
            local n = 0
            return function()
                n = n + 1
                if n <= 2 then return coroutine.yield("next"), n end
            end, t
        end})
        local s = ""
        for k, v in pairs(t) do s = s .. k .. v end
        return s
    end)
    print(co())
    --> =pairs
    print(co())
    --> =next
    print(co("a"))
    --> =next
    print(co("b"))
    --> =a1b2

    -- Yield from a __close handler run at the end of a block.
    co = coroutine.wrap(function()
        do
            local x <close> = setmetatable({}, {__close = function()
                coroutine.yield("closing")
            end})
        end
        return "done"
    end)
    print(co())
    --> =closing
    print(co())
    --> =done

    -- Yield with a debug hook set.
    local count = 0
    co = coroutine.create(function()
        debug.sethook(function() count = count + 1 end, "l")
        coroutine.yield("hooked")
        local x = 1
        return x
    end)
    print(coroutine.resume(co))
    --> =true	hooked
    print(coroutine.resume(co))
    --> =true	1
    print(count > 0)
    --> =true
end

-- Light coroutines resume each other and are closed like the others.
do
    -- A chain of coroutines each resuming the next.
    local function chain(n)
        if n == 0 then
            return coroutine.wrap(function() return "bottom" end)
        end
        local inner = chain(n - 1)
        return coroutine.wrap(function() return inner() end)
    end
    print(chain(50)())
    --> =bottom

    -- Closing a coroutine that has not started, and one that has.
    local co = coroutine.create(print)
    print(coroutine.close(co), coroutine.status(co))
    --> =true	dead
    co = coroutine.create(function() coroutine.yield() end)
    coroutine.resume(co)
    print(coroutine.close(co), coroutine.status(co))
    --> =true	dead
    print(coroutine.resume(co))
    --> ~false\t.*cannot resume dead

    -- A coroutine resuming itself or its resumer.
    co = coroutine.create(function() return coroutine.resume(co) end)
    print(coroutine.resume(co))
    --> ~true\tfalse\t.*cannot resume running thread
    local outer
    outer = coroutine.wrap(function()
        local inner = coroutine.wrap(function() return pcall(outer) end)
        return inner()
    end)
    print(outer())
    --> ~false\t.*cannot resume running thread

    -- Errors end the coroutine and keep their message.
    co = coroutine.create(function() coroutine.yield() error("oops") end)
    coroutine.resume(co)
    print(coroutine.resume(co))
    --> ~false\t.*oops
    print(coroutine.status(co))
    --> =dead

    -- Many suspended coroutines cost little: a goroutine each would not.
    local cos = {}
    for i = 1, 20000 do
        cos[i] = coroutine.wrap(function() coroutine.yield(i) return -i end)
        cos[i]()
    end
    local sum = 0
    for i = 1, #cos do sum = sum + cos[i]() end
    print(sum)
    --> =-200010000
end

-- Resumes of light coroutines nest on the Go stack, so there is a limit.
do
    local function nest(n)
        return coroutine.wrap(function() return n == 0 and 0 or 1 + nest(n - 1)() end)
    end
    print(pcall(nest(100)))
    --> =true	100
    print(pcall(nest(100000)))
    --> ~false\t.*stack overflow
end

-- A yield cannot cross a library function that calls Lua, as in Lua: it is
-- an error where the yield is, and coroutine.isyieldable says so.  pcall,
-- xpcall and dofile are the exceptions.
do
    local function try(f)
        local co = coroutine.create(f)
        local ok, msg = coroutine.resume(co)
        if ok then return coroutine.status(co) end
        return msg:match(":%d+: (.*)")
    end
    local yield = coroutine.yield
    local ts = setmetatable({}, {__tostring = function() yield() return "x" end})
    local idx = setmetatable({}, {__index = function(_, i) yield() return i < 3 and i or nil end})

    print(try(function() table.sort({3, 1, 2}, function(a, b) yield() return a < b end) end))
    --> =attempt to yield across a Go-call boundary
    print(try(function() tostring(ts) end))
    --> =attempt to yield across a Go-call boundary
    print(try(function() string.format("%s", ts) end))
    --> =attempt to yield across a Go-call boundary
    print(try(function() string.gsub("abc", "%w", function() yield() end) end))
    --> =attempt to yield across a Go-call boundary
    print(try(function() for i, v in ipairs(idx) do end end))
    --> =attempt to yield across a Go-call boundary
    print(try(function() table.unpack(idx, 1, 2) end))
    --> =attempt to yield across a Go-call boundary
    print(try(function() load(function() yield() end) end))
    --> =dead

    -- The same from further down, and when something catches the error.
    print(try(function()
        table.sort({2, 1}, function(a, b)
            print(pcall(pcall, yield))
            --> ~true\tfalse\t.*attempt to yield across a Go-call boundary
            return a < b
        end)
        yield()
    end))
    --> =suspended

    -- Another coroutine can yield there: the yield does not cross the sort.
    print(try(function()
        table.sort({2, 1}, function(a, b)
            local inner = coroutine.wrap(function() yield(1) yield(2) end)
            return inner() < inner()
        end)
    end))
    --> =dead

    -- isyieldable knows.
    print(try(function()
        print(coroutine.isyieldable())
        --> =true
        table.sort({2, 1}, function(a, b)
            print(coroutine.isyieldable())
            --> =false
            return a < b
        end)
        print(pcall(coroutine.isyieldable))
        --> =true	true
        print(coroutine.isyieldable())
        --> =true
    end))
    --> =dead

    -- Values to close in a function called by a library function are
    -- closed when an error goes through them, also when nothing catches it
    -- and the coroutine dies of it.
    local log = {}
    local closing = setmetatable({}, {__close = function(_, err)
        log[#log + 1] = "closed: " .. tostring(err):match(":%d+: (.*)")
    end})
    print(try(function()
        tostring(setmetatable({}, {__tostring = function()
            local x <close> = closing
            error("in tostring")
        end}))
    end))
    --> =in tostring
    print(table.concat(log, " "))
    --> =closed: in tostring
end
