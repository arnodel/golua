-- By default runtime.shouldstop returns false
print(runtime.shouldstop())
--> =false

-- runtime.shouldstop returns true if a cpu soft limit has been reached
print(runtime.callcontext({softlimits={cpu=100}}, function()
    print(runtime.shouldstop())
    --> =false
    local ctx = runtime.context()
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
    while not runtime.shouldstop() do end
    print(ctx.used.time >= 20, ctx.used.time <= 30)
    --> =true	true
end))
--> =done
