local quota = require 'quota'

-- string.pack
do
    local s = "a"
    local s1000 = ("a"):rep(1000)

    -- string.pack uses memory to store its output
    local ok = quota.rcall(0, 4000, string.pack, "ssss", s1000, s, s, s)
    print(ok)
    --> =true

    print(quota.rcall(0, 4000, string.pack, "ssss", s1000, s1000, s1000, s1000))
    --> =false

    -- string.pack uses cpu to produce its output
    ok = quota.rcall(400, 0, string.pack, "ssss", s1000, s, s, s)
    print(ok)
    --> =true

    print(quota.rcall(400, 0, string.pack, "ssss", s1000, s1000, s1000, s1000))
    --> =false
end

-- string.unpack
do
    local fmt = "i"
    local packed = string.pack(fmt, 100)

    print(string.unpack(fmt:rep(5), packed:rep(5)))
    --> ~100	100	100	100	100	.*

    -- string.unpack uses memory to produce its output
    local ok = quota.rcall(0, 1000, string.unpack, fmt:rep(20), packed:rep(20))
    print(ok)
    --> =true

    print(quota.rcall(0, 1000, string.unpack, fmt:rep(100), packed:rep(100)))
    --> =false

    -- string.unpack uses cpu to produce its output
    local ok = quota.rcall(100, 0, string.unpack, fmt:rep(50), packed:rep(50))
    print(ok)
    --> =true

    print(quota.rcall(100, 0, string.unpack, fmt:rep(500), packed:rep(500)))
    --> =false

    local fmt = "s"
    local packed10 = string.pack(fmt, ("a"):rep(10))
    local packed1000 = string.pack(fmt, ("a"):rep(1000))

    -- big strings need lots of memory
    local ok = quota.rcall(0, 2000, string.unpack, fmt:rep(20), packed10:rep(20))
    print(ok)
    --> =true

    print(quota.rcall(0, 2000, string.unpack, fmt:rep(20), packed1000:rep(20)))
    --> =false
end