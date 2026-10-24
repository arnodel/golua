-- Memory charged by calling f 10 times
local function used(f)
    local ctx = runtime.callcontext({kill={memory=100000}}, function()
        for i = 1, 10 do f() end
    end)
    return ctx.used.memory or 0
end

-- debug.getinfo charges the table it returns, like the equivalent table
-- constructor does.  All three arguments are passed because calling a Go
-- function with fewer arguments than it declares is charged a few bytes more,
-- which would hide the comparison.
local co = coroutine.running()
print(used(function() return debug.getinfo(co, 1, "") end) == used(function()
    return {name="", currentline=0, source=""}
end))
--> =true
