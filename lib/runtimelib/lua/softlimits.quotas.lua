-- By default runtime.shouldstop returns false
print(runtime.shouldstop())
--> =false

-- runtime.shouldstop returns true if a cpu soft limit has been reached
print(runtime.callcontext({softlimits={cpu=100}}, function()
    print(runtime.shouldstop())
    --> =false
    local ctx = runtime.context()
    print(ctx.softlimits.cpu)
    --> =100
    while not runtime.shouldstop() do end
    print(ctx.used.cpu >= 100, ctx.used.cpu <= 200)
    --> =true	true
end))
--> =done

-- runtime.shouldstop returns true if a mem soft limit has been reached
print(runtime.callcontext({softlimits={mem=1000}}, function()
    print(runtime.shouldstop())
    --> =false
    local ctx = runtime.context()
    print(ctx.softlimits.mem)
    --> =1000
    local a = "x"
    while not runtime.shouldstop() do 
        a = a .. a -- consume some memory
    end
    print(ctx.used.mem >= 1000, ctx.used.mem <= 2000)
    --> =true	true
end))
--> =done

-- runtime.shouldstop returns true if a time soft limit has been reached
print(runtime.callcontext({softlimits={time=20}}, function()
    print(runtime.shouldstop())
    --> =false
    local ctx = runtime.context()
    print(ctx.softlimits.time)
    --> =20
    while not runtime.shouldstop() do end
    print(ctx.used.time >= 20, ctx.used.time <= 30)
    --> =true	true
end))
--> =done

-- soft limits cannot exceed hard limits, either in the same context or in the
-- parent context
runtime.callcontext({limits={time=1000}}, function() 
    runtime.callcontext({softlimits={time=2000}}, function()
        print(runtime.context().softlimits.time <= 1000)
        --> =true
    end)
end)
runtime.callcontext({limits={time=1000}, softlimits={time=5000}}, function()
    print(runtime.context().softlimits.time <= 1000)
    --> =true
end)

-- soft limits can increase from the parent's soft limit.
runtime.callcontext({softlimits={cpu=1000}, limits={cpu=2000}}, function()
    runtime.callcontext({softlimits={cpu=3000}}, function()
        local l = runtime.context().softlimits.cpu
        print( l >= 1500, l <= 2000)
        --> =true	true
    end)
end)
