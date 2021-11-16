local quota = require'quota'

local s = "helLO"
local s1000 = s:rep(1000)

-- string.lower, string.upper, string.reverse consume memory
do
    print(quota.rcall(0, 4000, string.lower, s))
    --> =true	hello

    print(quota.rcall(0, 4000, string.lower, s1000))
    --> =false


    -- string.upper consumes memory

    print(quota.rcall(0, 4000, string.upper, s))
    --> =true	HELLO

    print(quota.rcall(0, 4000, string.upper, s1000))
    --> =false


    -- string.reverse consumes memory

    print(quota.rcall(0, 4000, string.reverse, s))
    --> =true	OLleh

    print(quota.rcall(0, 4000, string.reverse, s1000))
    --> =false
end

-- string.sub consumes memory
do
    print(quota.rcall(0, 1000, string.sub, s, 3, 2000))
    --> =true	lLO

    print(quota.rcall(0, 1000, string.sub, s1000, 3, 2000))
    --> =false
end

-- string.byte consumes memory
do
    -- helper function to consume the returned bytes from string.byte.
    function len(s)
        return select('#', s:byte(1, #s))
    end

    print(len("foobar"))
    --> =6

    print(quota.rcall(0, 10000, len, s1000))
    --> =false

    -- string.char consumes memory

    print(quota.rcall(0, 1000, string.char, s:byte(1, #s)))
    --> =true	helLO

    print(quota.rcall(0, 1000, string.char, s1000:byte(1, 1200)))
    --> =false


    -- string.rep consumes memory

    print(quota.rcall(0, 1000, string.rep, "ha", 10))
    --> =true	hahahahahahahahahaha

    print(quota.rcall(0, 1000, string.rep, "ha", 600))
    --> =false
end

-- string.dump consumes memory and cpu
do
    local function mk(n)
        return load(
            "x = 0\n" .. 
            ("x = x + 1"):rep(n, "\n") .. 
            "\nreturn x"
        )
    end

    print(mk(10)())
    --> =10

    -- A function with 12 lines is ok for CPU
    local ok = quota.rcall(1000, 0, string.dump, mk(10))
    print(ok)
    --> =true
    
    -- One with 500 lines runs out of CPU
    print(quota.rcall(1000, 0, string.dump, mk(500)))
    --> =false

    -- A function with 12 lines is ok for mem
    local ok = quota.rcall(0, 1000, string.dump, mk(10))
    print(ok)
    --> =true
    
    -- One with 500 lines runs out of mem
    print(quota.rcall(1000, 0, string.dump, mk(500)))
    --> =false
end

-- string.format consumes memory and cpu
do
    -- a long format needs to be scanned and uses cpu
    print(quota.rcall(1000, 0, string.format, ("a"):rep(50)))
    --> =true	aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa

    print(quota.rcall(1000, 0, string.format, ("a"):rep(1000)))
    --> =false

    -- format requires memory to build the formatted string
    local s = "aa"
    print(quota.rcall(0, 1000, string.format, "%s %s %s %s %s", s, s, s, s, s))
    --> =true	aa aa aa aa aa

    s = ("a"):rep(1000)
    print(quota.rcall(0, 5000, string.format, "%s %s %s %s %s", s, s, s, s, s))
    --> =false
end
