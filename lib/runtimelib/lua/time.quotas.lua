-- Test time bound contexts

-- A time bound context stops when the time is exceeded
local n = 0
local ctx = runtime.callcontext({kill={millis=10}}, function()
    local ctx = runtime.context()
    print(ctx.kill.millis, ctx.kill.seconds)
    --> =10	0.01
    while true do
        n = n + 1
    end
end)

-- The context was killed
print(ctx)
--> =killed

-- It lasted for at least 1e6ms
print(ctx.used.millis >= 10)
--> =true

-- It didn't last much more than that (could be flaky)
print(ctx.used.millis <= 15)
--> =true

-- Significant work was done while it lasted (could be flaky)
print(n > 20000)
--> =true

-- The outer context keeps track of time spent in the inner context
local ctx = runtime.callcontext({kill={seconds=0.1}}, function()
    for i = 1, 3 do
        runtime.callcontext({kill={millis=10}}, function()
            while true do end
        end)
    end
end)

print(ctx.used.millis >= 30)
--> =true

-- Nested contexts are bound by the time limit of their parent context.
local ctx = runtime.callcontext({kill={millis=10}}, function()
        runtime.callcontext({}, function ()
            print(runtime.context().kill.millis)
            --> =10
        end)
        runtime.callcontext({kill={seconds=1}}, function()
            print(runtime.context().kill.millis <= 10)
            --> =true
        end)
end)
