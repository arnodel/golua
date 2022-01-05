-- load tests
do
    -- If passed a function load consumes cpu and memory to build the string

    print(runtime.callcontext({kill={memory=10000}}, load, function() return "print('hello')\n" end))
    --> =killed

    print(runtime.callcontext({kill={cpu=10000}}, load, function() return "print('hello')\n" end))
    --> =killed

    -- Same if passed a big string

    print(runtime.callcontext({kill={memory=10000}}, load, ("print('hello')"):rep(1000, "\n")))
    --> =killed

    print(runtime.callcontext({kill={cpu=10000}}, load, ("print('hello')"):rep(10000, "\n")))
    --> =killed
end

-- loadfile tests
do
    -- loadfile consumes cpu and memory to load the string

    local ctx, m = runtime.callcontext({kill={memory=100000}}, loadfile, "lua/big.lua.notest")
    print(ctx, m())
    --> =done	hello

    print(runtime.callcontext({kill={memory=10000}}, loadfile, "lua/big.lua.notest"))
    --> =killed

    local ctx, m = runtime.callcontext({kill={cpu=10000}}, loadfile, "lua/big.lua.notest")
    print(ctx, m())
    --> =done	hello

    print(runtime.callcontext({kill={cpu=1000}}, loadfile, "lua/big.lua.notest"))
    --> =killed
end

-- dofile tests
do
    -- dofile consumes cpu and memory to load the string

    local ctx, m = runtime.callcontext({kill={memory=100000}}, dofile, "lua/big.lua.notest")
    print(ctx, m)
    --> =done	hello

    print(runtime.callcontext({kill={memory=10000}}, dofile, "lua/big.lua.notest"))
    --> =killed

    local ctx, m = runtime.callcontext({kill={cpu=10000}}, dofile, "lua/big.lua.notest")
    print(ctx, m)
    --> =done	hello

    print(runtime.callcontext({kill={cpu=1000}}, dofile, "lua/big.lua.notest"))
    --> =killed
end
