-- A context returned by runtime.context() keeps referring to that context, even
-- while other contexts run inside it, and after it has finished.

local saved
local ctx = runtime.callcontext({kill={cpu=1000000}}, function()
    saved = runtime.context()
    runtime.callcontext({kill={cpu=1000}}, function()
        print(saved.kill.cpu, runtime.context().kill.cpu)
        --> =1000000	1000

        print(saved.status)
        --> =live
    end)
end)
print(ctx, saved.status)
--> =done	done

-- Killing a context from a context running inside it kills that context, every
-- context in between (including those created by pcall) and the context
-- itself.  No more code runs in any of them.

local ran = {}
local ctx = runtime.callcontext({}, function()
    local outer = runtime.context()
    runtime.callcontext({}, function()
        pcall(function()
            outer:killnow()
            ran.inner = true
        end)
        ran.pcall = true
    end)
    ran.outer = true
end)
print(ctx, ran.inner, ran.pcall, ran.outer)
--> =killed	nil	nil	nil

-- The same with runtime.killcontext(ctx)

local ran = false
local ctx = runtime.callcontext({}, function()
    local outer = runtime.context()
    runtime.callcontext({}, function()
        runtime.killcontext(outer)
    end)
    ran = true
end)
print(ctx, ran)
--> =killed	false

-- The enclosing context carries on
print(runtime.context())
--> =live

-- Stopping a context from a context running inside it makes it due once it is
-- current again, without affecting the inner context.

runtime.callcontext({}, function()
    local outer = runtime.context()
    runtime.callcontext({}, function()
        outer:stopnow()
        print(runtime.contextdue(), outer.due)
        --> =false	true
    end)
    print(runtime.contextdue())
    --> =true
end)

-- Killing a context that has finished has no effect

local done = runtime.callcontext({}, function() end)
runtime.killcontext(done)
done:killnow()
print(done, "still running")
--> =done	still running
