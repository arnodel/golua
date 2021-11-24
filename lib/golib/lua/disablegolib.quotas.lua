

-- golib not cpu safe
runtime.callcontext({cpulimit=10000}, function()
    local ctx = runtime.context()

    print(pcall(double, 2))
    --> =false	missing flags: cpusafe

    print(pcall(function() return polly.Age end))
    --> =false	missing flags: cpusafe

    print(pcall(golib.import, "fmt"))
    --> =false	missing flags: cpusafe
end)

-- golib not memory safe
runtime.callcontext({memlimit=10000}, function()
    local ctx = runtime.context()

    print(pcall(double, 2))
    --> =false	missing flags: memsafe

    print(pcall(function() return polly.Age end))
    --> =false	missing flags: memsafe

    print(pcall(golib.import, "fmt"))
    --> =false	missing flags: memsafe
end)

-- golib not io safe
runtime.callcontext({flags="iosafe"}, function()
    local ctx = runtime.context()

    print(pcall(double, 2))
    --> =false	missing flags: iosafe

    print(pcall(function() return polly.Age end))
    --> =false	missing flags: iosafe

    print(pcall(golib.import, "fmt"))
    --> =false	missing flags: iosafe
end)
