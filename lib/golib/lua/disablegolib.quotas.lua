

-- golib not cpu safe
runtime.callcontext({kill={cpu=10000}}, function()
    local ctx = runtime.context()

    print(pcall(double, 2))
    --> ~false\t.*: missing flags: cpusafe

    print(pcall(function() return polly.Age end))
    --> ~false\t.*: missing flags: cpusafe

    print(pcall(golib.import, "fmt"))
    --> ~false\t.*: missing flags: cpusafe
end)

-- golib not memory safe
runtime.callcontext({kill={memory=10000}}, function()
    local ctx = runtime.context()

    print(pcall(double, 2))
    --> ~false\t.*: missing flags: memsafe

    print(pcall(function() return polly.Age end))
    --> ~false\t.*: missing flags: memsafe

    print(pcall(golib.import, "fmt"))
    --> ~false\t.*: missing flags: memsafe
end)

-- golib not io safe
runtime.callcontext({flags="iosafe"}, function()
    local ctx = runtime.context()

    print(pcall(double, 2))
    --> ~false\t.*: missing flags: iosafe

    print(pcall(function() return polly.Age end))
    --> ~false\t.*: missing flags: iosafe

    print(pcall(golib.import, "fmt"))
    --> ~false\t.*: missing flags: iosafe
end)
