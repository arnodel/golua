-- Memory charged by calling f 10 times
local function used(f)
    local ctx = runtime.callcontext({kill={memory=100000}}, function()
        for i = 1, 10 do f() end
    end)
    return ctx.used.memory or 0
end

-- os.date("*t") charges the table it returns, like the equivalent table
-- constructor does.
print(used(function() return os.date("!*t", 0) end) == used(function()
    return {year=1970, month=1, day=1, hour=0, min=0, sec=0, wday=5, yday=1, isdst=false}
end))
--> =true
