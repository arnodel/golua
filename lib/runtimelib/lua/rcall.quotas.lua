-- runtime.callcontext(contextDef, f, [arg1, ...])
--
-- The runtime.callcontext function creates a new context from the contextDef
-- and runs f(arg1, ...). It always returns the context in which f was run.  If
-- that function can be executed within the resource limits in the context, it
-- also returns its return values.
local ctx, output = runtime.callcontext({limits={mem=1000, cpu=1000}}, function() return 1 end)
print(output)
--> =1

-- The returned context can be inspected but is "frozen" (no longer active)
print(ctx.status, ctx.limits.mem, ctx.limits.cpu)
--> =done	1000	1000
print(ctx.used.cpu > 0, ctx.used.mem > 0)
--> =true	true

-- If the function called by callcontext errors, the status reflects this and
-- the error is returned.  So callcontext "implements" pcall.
print(runtime.callcontext({}, error, "an error"))
--> =error	an error

-- runtime.context()
--
-- runtime.context returns the current execution context.  It is not possible to
-- mutate this context but it is a "live" reference.

runtime.callcontext({limits={mem=10000, cpu=10000}}, function()
    local ctx = runtime.context()
    print(ctx)
    --> =live

    local cpu = ctx.used.cpu
    for i = 1, 100 do end
    print(ctx.used.cpu - cpu >= 100)
    --> =true

    local mem = ctx.used.mem
    s = ("a"):rep(1000)
    print(ctx.used.mem - mem >= 1000)
    --> =true
end)

-- runtime.cancelcontext()
--
-- The runtime.cancelcontext function terminates the current context.

print(runtime.callcontext({}, function()
    print("before cancel")
    --> =before cancel
    runtime.cancelcontext()
    print("after cancel")
    -- Not reached
end))
--> =killed


-- Here the passed in function is an infinite loop, so execution will stop when
-- the budget of 1000 cpu is consumed.
local x = 0
local ctx = runtime.callcontext({limits={cpu=1000}}, function() while true do x = x + 1 end end)

-- When the function reached a limit the returned context has a "killed" status
print(ctx.status)
--> =killed

-- Here we reached the cpu limit
print(ctx.used.cpu >= 990, ctx.limits.cpu)
--> =true	1000

-- If a resource is not limited its limit reported as nil
print(ctx.limits.mem)
--> =nil

-- Check that runtime managed to do a few iterations before being terminated
print(x > 10)
--> =true

-- Helper function for checking limits below.
function getCurrentLimits()
    local limits = runtime.context().limits
    return limits.cpu, limits.mem
end

print(getCurrentLimits())
--> =nil	nil

-- Calls to callcontext can be nested but it is not possible to increase the
-- resource available in a context.  A context is always created "as a child" of
-- the current context.
runtime.callcontext({limits={cpu=100000, mem=200000}}, function()
    local ctx = runtime.context()
    print(getCurrentLimits())
    --> =100000	200000

    -- It's not possible to increase the quotas
    runtime.callcontext({limits={cpu=20000, mem=30000}}, function()
        local limits = runtime.context().limits
        print(limits.cpu <= 100000, limits.mem <= 200000)
        --> =true	true
    end)

    -- CPU consumed inside the callcontext is accounted for once the callcontext has
    -- finished.
    local cpu = ctx.used.cpu

    -- It's possible to further decrease the quotas
    print(runtime.callcontext({limits={cpu=5000, mem=5000}}, function()
        local limits = runtime.context().limits
        print(limits.cpu, limits.mem)
        --> =5000	5000
        while true do end
    end))
    --> =killed

    print(ctx.used.cpu - cpu >= 5000)
    --> =true

    -- Quotas get reset after
    print(getCurrentLimits())
    --> =100000	200000

    -- Memory consumed inside the callcontext is accounted for once the callcontext has
    -- finished.
    local mem = ctx.used.mem 

    runtime.callcontext({limits={cpu=10000, mem=20000}}, function()
        -- Consume some memory to check that will be accounted for outside callcontext
        local s = ("a"):rep(10000)
    end)
    print(ctx.used.mem - mem >= 10000)
    --> =true
end)

-- Quotas get reset to their initial value
print(getCurrentLimits())
--> =nil	nil
