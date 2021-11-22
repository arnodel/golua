local ctx = runtime.context()
print(ctx.golib)
--> =on

runtime.callcontext({golib="off"}, function()
    local ctx = runtime.context()
    print(ctx.golib)
    --> =off
    print(pcall(double, 2))
    --> =false	go disabled

    print(pcall(function() return polly.Age end))
    --> =false	go disabled

    print(pcall(golib.import, "fmt"))
    --> =false	go disabled

    -- Go can't be turned back on
    runtime.callcontext({golib="on"}, function()
        local ctx = runtime.context()
        print(ctx.golib)
        --> =off
        print(pcall(double, 2))
        --> =false	go disabled
    end)
end)

print(ctx.golib)
--> =on
