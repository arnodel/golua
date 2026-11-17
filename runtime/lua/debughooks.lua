-- Test debug hooks
do
    local function bar() 
        local x = 1
    end
    local function foo()
        return bar()
    end
    local cofoo = coroutine.create(foo)
    debug.sethook(cofoo, print, "clr")
    coroutine.resume(cofoo)
    --> =call
    --> =line	7
    --> =tail call
    --> =line	4
    --> =return
    debug.sethook(cofoo)
end

-- A line hook that asks where the function it interrupted is gets the line it
-- was called for, on every line and not only on the first.
do
    local seen = {}
    local function f()
        local a = 1
        local b = 2
        local c = 3
        return a + b + c
    end
    local base = debug.getinfo(1, "l").currentline
    debug.sethook(function(event, line)
        seen[#seen + 1] = (line - base) .. "=" .. (debug.getinfo(2, "l").currentline - base)
    end, "l")
    f()
    debug.sethook()
    print(table.concat(seen, " "))
    --> =4=4 -5=-5 -4=-4 -3=-3 -2=-2 5=5

    -- So does a traceback made by the hook.
    local wrong = 0
    debug.sethook(function(event, line)
        local top = debug.traceback("", 2):match(":(%d+)%)")
        if tonumber(top) ~= line then wrong = wrong + 1 end
    end, "l")
    local x = 1
    local y = 2
    local z = 3
    debug.sethook()
    print(wrong)
    --> =0
end

-- A hook set on a coroutine from outside it is called from then on, with no
-- event missing and none made up, whatever the coroutine was doing.
do
    local function events(setup)
        local log = {}
        local function hook(ev) log[#log + 1] = ev end
        setup(hook)
        return table.concat(log, " ")
    end

    -- Before its first resume: the first event is the call of its body.
    print(events(function(hook)
        local co = coroutine.create(function()
            local function f() end
            f()
            debug.sethook()
        end)
        debug.sethook(co, hook, "cr")
        coroutine.resume(co)
    end))
    --> =call call return call

    -- Suspended in a yield: the first event is the return of the yield.
    print(events(function(hook)
        local co = coroutine.create(function()
            coroutine.yield()
            local function f() end
            f()
            debug.sethook()
        end)
        coroutine.resume(co)
        debug.sethook(co, hook, "cr")
        coroutine.resume(co)
    end))
    --> =return call return call

    -- Waiting for a coroutine it has resumed, which sets the hook: the first
    -- event is the return of that resume.
    print(events(function(hook)
        local co
        co = coroutine.create(function()
            coroutine.wrap(function() debug.sethook(co, hook, "cr") end)()
            local function f() end
            f()
            debug.sethook()
        end)
        coroutine.resume(co)
    end))
    --> =return call return call
end

-- A hook cannot yield: it is called from the middle of what the coroutine
-- is doing.
do
    local msg
    local co = coroutine.create(function()
        debug.sethook(function()
            debug.sethook()
            msg = select(2, pcall(coroutine.yield))
        end, "l")
        local x = 1
        return "done"
    end)
    print(coroutine.resume(co))
    --> =true	done
    print(msg)
    --> ~attempt to yield across a Go-call boundary
end

-- The events of a coroutine that yields, hooked from before its start.  The
-- call of yield is one call and one return, however the coroutine is
-- suspended and resumed in between.
do
    local function trace(body, mask, resumes)
        local events = {}
        local co = coroutine.create(body)
        debug.sethook(co, function(ev) events[#events + 1] = ev end, mask)
        for i = 1, resumes do coroutine.resume(co) end
        return table.concat(events, " ")
    end

    -- One yield: call of the body, call and return of yield, return of the
    -- body.
    print(trace(function()
        coroutine.yield(10)
        return 20
    end, "cr", 2))
    --> =call call return return

    -- With lines, as the Lua test suite has it.
    print(trace(function()
        coroutine.yield(10)
        return 20
    end, "clr", 2))
    --> =call line call return line return

    -- Three yields in a row, the last one never resumed.
    print(trace(function()
        coroutine.yield()
        coroutine.yield()
        coroutine.yield()
    end, "cr", 3))
    --> =call call return call return call

    -- A yield from a function the body calls.
    print(trace(function()
        local function f() coroutine.yield() end
        f()
    end, "cr", 2))
    --> =call call call return return return

    -- A coroutine made by wrap, hooked from inside itself.
    local events = {}
    local gen = coroutine.wrap(function()
        debug.sethook(function(ev) events[#events + 1] = ev end, "r")
        coroutine.yield()
        coroutine.yield()
        debug.sethook()
    end)
    gen() gen() gen()
    print(table.concat(events, " "))
    --> =return return return
end
