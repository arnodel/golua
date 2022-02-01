-- By default runtime.contextdue returns false
print(runtime.contextdue())
--> =false

-- runtime.contextdue returns true if a cpu soft limit has been reached
print(runtime.callcontext({stop={cpu=100}}, function()
    print(runtime.contextdue())
    --> =false
    local ctx = runtime.context()
    print(ctx.stop.cpu)
    --> =100
    while not runtime.contextdue() do end
    print(ctx.used.cpu >= 100, ctx.used.cpu <= 200)
    --> =true	true
end))
--> =done

-- runtime.contextdue returns true if a mem soft limit has been reached
print(runtime.callcontext({stop={memory=1000}}, function()
    print(runtime.contextdue())
    --> =false
    local ctx = runtime.context()
    print(ctx.stop.memory)
    --> =1000
    local a = "x"
    while not runtime.contextdue() do 
        a = a .. a -- consume some memory
    end
    print(ctx.used.memory >= 1000, ctx.used.memory <= 2000)
    --> =true	true
end))
--> =done

-- runtime.contextdue returns true if a time soft limit has been reached
print(runtime.callcontext({stop={millis=20}}, function()
    print(runtime.contextdue())
    --> =false
    local ctx = runtime.context()
    print(ctx.stop.millis)
    --> =20
    while not runtime.contextdue() do end
    print(ctx.used.millis >= 20, ctx.used.millis <= 30)
    --> =true	true
end))
--> =done

-- soft limits cannot exceed hard limits, either in the same context or in the
-- parent context
runtime.callcontext({kill={millis=1000}}, function() 
    runtime.callcontext({stop={millis=2000}}, function()
        print(runtime.context().stop.millis <= 1000)
        --> =true
    end)
end)
runtime.callcontext({kill={millis=1000}, stop={millis=5000}}, function()
    print(runtime.context().stop.millis <= 1000)
    --> =true
end)

-- soft limits cannot increase from the parent's soft limit.
runtime.callcontext({stop={cpu=1000}, kill={cpu=2000}}, function()
    runtime.callcontext({stop={cpu=3000}}, function()
        print(runtime.context().stop.cpu)
        --> =1000
    end)
end)
