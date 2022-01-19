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
