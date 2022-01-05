-- string.pack
do
    local s = "a"
    local s1000 = ("a"):rep(1000)

    -- string.pack uses memory to store its output
    local ctx = runtime.callcontext({kill={memory=4000}}, string.pack, "ssss", s1000, s, s, s)
    print(ctx)
    --> =done

    print(runtime.callcontext({kill={memory=4000}}, string.pack, "ssss", s1000, s1000, s1000, s1000))
    --> =killed

    -- string.pack uses cpu to produce its output
    ctx = runtime.callcontext({kill={cpu=400}}, string.pack, "ssss", s1000, s, s, s)
    print(ctx)
    --> =done

    print(runtime.callcontext({kill={cpu=400}}, string.pack, "ssss", s1000, s1000, s1000, s1000))
    --> =killed
end

-- string.unpack
do
    local fmt = "i"
    local packed = string.pack(fmt, 100)

    print(string.unpack(fmt:rep(5), packed:rep(5)))
    --> ~100	100	100	100	100	.*

    -- string.unpack uses memory to produce its output
    local ctx = runtime.callcontext({kill={memory=1000}}, string.unpack, fmt:rep(20), packed:rep(20))
    print(ctx)
    --> =done

    print(runtime.callcontext({kill={memory=1000}}, string.unpack, fmt:rep(100), packed:rep(100)))
    --> =killed

    -- string.unpack uses cpu to produce its output
    local ctx = runtime.callcontext({kill={cpu=100}}, string.unpack, fmt:rep(50), packed:rep(50))
    print(ctx)
    --> =done

    print(runtime.callcontext({kill={cpu=100}}, string.unpack, fmt:rep(500), packed:rep(500)))
    --> =killed

    local fmt = "s"
    local packed10 = string.pack(fmt, ("a"):rep(10))
    local packed1000 = string.pack(fmt, ("a"):rep(1000))

    -- big strings need lots of memory
    local ctx = runtime.callcontext({kill={memory=2000}}, string.unpack, fmt:rep(20), packed10:rep(20))
    print(ctx)
    --> =done

    print(runtime.callcontext({kill={memory=2000}}, string.unpack, fmt:rep(20), packed1000:rep(20)))
    --> =killed
end