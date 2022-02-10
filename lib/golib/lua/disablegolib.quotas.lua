local function checkflag(flag, f, ...)
    if f then
        local st, err = pcall(f, ...)
        if st or not string.match(err, flag, 1, true) then
            print(flag, "not found:", err)
            return
        end
    end
    print("ok")
end

-- golib not cpu safe
runtime.callcontext({kill={cpu=10000}}, function()
    local ctx = runtime.context()

    checkflag("cpusafe", double, 2)
    --> =ok

    checkflag("cpusafe", function() return polly.Age end)
    --> =ok

    checkflag("cpusafe", golib.import, "fmt")
    --> =ok
end)

-- golib not memory safe
runtime.callcontext({kill={memory=10000}}, function()
    local ctx = runtime.context()

    checkflag("memsafe", double, 2)
    --> =ok

    checkflag("memsafe", function() return polly.Age end)
    --> =ok

    checkflag("memsafe", golib.import, "fmt")
    --> =ok
end)

-- golib not io safe
runtime.callcontext({flags="iosafe"}, function()
    local ctx = runtime.context()

    checkflag("iosafe", double, 2)
    --> =ok

    checkflag("iosafe", function() return polly.Age end)
    --> =ok

    checkflag("iosafe", golib.import, "fmt")
    --> =ok
end)
