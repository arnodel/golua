-- Test time bound contexts

-- A time bound context stops when the time is exceeded
local n = 0
local ctx = runtime.callcontext({kill={millis=100}}, function()
    local ctx = runtime.context()
    print(ctx.kill.millis, ctx.kill.seconds)
    --> =100.0	0.1
    while true do
        n = n + 1
    end
end)

-- The context was killed
print(ctx)
--> =killed

-- It lasted for at least 100ms
print(ctx.used.millis >= 100)
--> =true

-- It didn't last much more than that.  The limit is normally exceeded by a few
-- milliseconds at most, but on a loaded machine the process can be descheduled
-- across the deadline and it is checked only when execution resumes, so the
-- bound needs to be generous.
print(ctx.used.millis < 1000)
--> =true

-- Significant work was done while it lasted
print(n > 10000)
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

-- Nested contexts are bound by the time limit of their parent context, minus
-- the time the parent has already used (which may be a few milliseconds).
local ctx = runtime.callcontext({kill={millis=10}}, function()
        runtime.callcontext({}, function ()
            local millis = runtime.context().kill.millis
            print(millis > 0 and millis <= 10)
            --> =true
        end)
        runtime.callcontext({kill={seconds=1}}, function()
            print(runtime.context().kill.millis <= 10)
            --> =true
        end)
end)
