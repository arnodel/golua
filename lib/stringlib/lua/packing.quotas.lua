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
    local ok = quota.rcall(400, 0, string.pack, "ssss", s1000, s, s, s)
    print(ok)
    --> =true

    print(quota.rcall(400, 0, string.pack, "ssss", s1000, s1000, s1000, s1000))
    --> =false
end