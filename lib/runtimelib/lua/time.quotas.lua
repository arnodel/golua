-- Test time bound contexts (time is in ms)

-- A time bound context stops when the time is exceeded
local n = 0
local ctx = runtime.callcontext({limits={time=10}}, function()
    local ctx = runtime.context()
    print(ctx.limits.time)
    --> =10
    while true do
        n = n + 1
    end
end)

-- The context was killed
print(ctx)
--> =killed

-- It lasted for at least 1e6ms
print(ctx.used.time >= 10)
--> =true

-- It didn't last much more than that (could be flaky)
print(ctx.used.time <= 11)
--> =true

-- Significant work was done while it lasted (could be flaky)
print(n > 50000)
--> =true

-- The outer context keeps track of time spent in the inner context
local ctx = runtime.callcontext({limits={time=100}}, function()
    for i = 1, 3 do
        runtime.callcontext({limits={time=10}}, function()
            while true do end
        end)
    end
end)

print(ctx.used.time >= 30)
--> =true

-- Nested contexts are bound by the time limit of their parent context.
local ctx = runtime.callcontext({limits={time=10}}, function()
        runtime.callcontext({}, function ()
            print(runtime.context().limits.time)
            --> =10
        end)
        runtime.callcontext({limits={time=1000}}, function()
            print(runtime.context().limits.time)
            --> =10
        end)
end)
