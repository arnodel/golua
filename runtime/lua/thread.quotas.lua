local ctx, co = runtime.callcontext({kill={cpu=1000}}, coroutine.create, function (n)
    while true do
        local d = 0
        while n > 0 do
            n = n - 1
            d = d + 2
        end
        n = coroutine.yield(d)
    end
end)

-- coroutines are resumed in the context of the resume call, see below: the cpu
-- limit of 10000 is applied on each resume call

print(runtime.callcontext({kill={cpu=10000}}, coroutine.resume, co, 100))
--> =done	true	200

print(runtime.callcontext({kill={cpu=10000}}, coroutine.resume, co, 500))
--> =done	true	1000

print(runtime.callcontext({kill={cpu=10000}}, coroutine.resume, co, 500))
--> =done	true	1000

-- 2000 iterations of the loop cost more than 10000 cpu (each costs 7 opcodes
-- since the compiler stopped copying registers for assignments; it was 9).
print(runtime.callcontext({kill={cpu=10000}}, coroutine.resume, co, 2000))
--> =killed

-- If a coroutine ran out of resources, then it becomes dead and it cannot be resumed

print(coroutine.status(co))
--> =dead

print(coroutine.resume(co, 100))
--> =false	cannot resume dead thread

-- A coroutine started outside a context with a memory limit can end inside
-- it.  It releases its stack there, more memory than the context has used so
-- far: that used to crash the whole program.

local co = coroutine.create(function() coroutine.yield(1) end)
coroutine.resume(co)
print(runtime.callcontext({kill={memory=1000000}}, coroutine.resume, co))
--> =done	true

print(coroutine.status(co))
--> =dead

-- The same from inside a sandbox, which can't otherwise reach outside it.

print(runtime.callcontext({kill={memory=10000000, cpu=1000000}, flags="memsafe cpusafe iosafe"}, function()
    local co = coroutine.create(function() coroutine.yield(1) end)
    coroutine.resume(co)
    return runtime.callcontext({kill={memory=1000000}}, coroutine.resume, co)
end))
--> =done	done	true
