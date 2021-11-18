-- runtime.callcontext(contextDef, f, [arg1, ...])
--
-- The runtime.callcontext function creates a new context from the contextDef
-- and runs f(arg1, ...). It always returns the context in which f was run.  If
-- that function can be executed within the resource limits in the context, it
-- also returns its return values.
local ctx, output = runtime.callcontext({memlimit=1000, cpulimit=1000}, function() return 1 end)
print(output)
--> =1

-- The returned context can be inspected but is "frozen" (no longer active)
print(ctx.status, ctx.memlimit, ctx.cpulimit)
--> =done	1000	1000
print(ctx.cpuused > 0, ctx.memused > 0)
--> =true	true

-- runtime.context()
--
-- runtime.context returns the current execution context.  It is not possible to
-- mutate this context but it is a "live" reference.

runtime.callcontext({memlimit=10000, cpulimit=10000}, function()
    local ctx = runtime.context()
    print(ctx)
    --> =live

    local cpu = ctx.cpuused
    for i = 1, 100 do end
    print(ctx.cpuused - cpu >= 100)
    --> =true

    local mem = ctx.memused
    s = ("a"):rep(1000)
    print(ctx.memused - mem >= 1000)
    --> =true
end)



-- Here the passed in function is an infinite loop, so execution will stop when
-- the budget of 1000 cpu is consumed.
local x = 0
local ctx = runtime.callcontext({cpulimit=1000}, function() while true do x = x + 1 end end)

-- When the function reached a limit the returned context has a "killed" status
print(ctx.status)
--> =killed

-- Here we reached the cpu limit
print(ctx.cpuused, ctx.cpulimit)
--> =1000	1000

-- If a resource is not limited its limit reported as nil
print(ctx.memlimit)
--> =nil

-- Check that runtime managed to do a few iterations before being terminated
print(x > 10)
--> =true

-- Helper function for checking limits below.
function getCurrentLimits()
    local ctx = runtime.context()
    return ctx.cpulimit, ctx.memlimit
end

print(getCurrentLimits())
--> =nil	nil

-- Calls to callcontext can be nested but it is not possible to increase the
-- resource available in a context.  A context is always created "as a child" of
-- the current context.
runtime.callcontext({cpulimit=10000, memlimit=20000}, function()
    local ctx = runtime.context()
    print(getCurrentLimits())
    --> =10000	20000

    -- It's not possible to increase the quotas
    runtime.callcontext({cpulimit=20000, memlimit=30000}, function()
        local ctx = runtime.context()
        print(ctx.cpulimit <= 10000, ctx.memlimit <= 20000)
        --> =true	true
    end)

    -- CPU consumed inside the callcontext is accounted for once the callcontext has
    -- finished.
    local cpu = ctx.cpuused

    -- It's possible to further decrease the quotas
    print(runtime.callcontext({cpulimit=5000, memlimit=5000}, function()
        local ctx = runtime.context()
        print(ctx.cpulimit, ctx.memlimit)
        --> =5000	5000
        while true do end
    end))
    --> =killed

    print(ctx.cpuused - cpu >= 5000)
    --> =true

    -- Quotas get reset after
    print(getCurrentLimits())
    --> =10000	20000

    -- Memory consumed inside the callcontext is accounted for once the callcontext has
    -- finished.
    local mem = ctx.memused

    runtime.callcontext({cpulimit=10000, memlimit=20000}, function()
        -- Consume some memory to check that will be accounted for outside callcontext
        local s = ("a"):rep(10000)
    end)
    print(ctx.memused - mem >= 10000)
    --> =true
end)

-- Quotas get reset to their initial value
print(getCurrentLimits())
--> =nil	nil
