
-- Test time bound contexts (time is in ns, so 1e7 = 10ms)
local n = 0
local ctx = runtime.callcontext({limits={time=1e7}}, function()
    local ctx = runtime.context()
    print(ctx.limits.time)
    --> =10000000
    while true do
        n = n + 1
    end
end)

-- the context was killed
print(ctx)
--> =killed

-- It lasted for at least 1e6ms
print(ctx.used.time >= 1e7)
--> =true

-- It didn't last much more than that
print(ctx.used.time <= 1.01e7)
--> =true

-- Significant work was done while it lasted (could be flaky)
print(n > 50000)
--> =true
