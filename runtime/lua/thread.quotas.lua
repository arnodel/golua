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

print(runtime.callcontext({kill={cpu=10000}}, coroutine.resume, co, 1000))
--> =killed

-- If a coroutine ran out of resources, then it becomes dead and it cannot be resumed

print(coroutine.status(co))
--> =dead

print(coroutine.resume(co, 100))
--> =false	cannot resume dead thread
