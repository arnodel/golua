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
