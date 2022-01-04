local s = "helLO"
local s1000 = s:rep(1000)

-- string.lower, string.upper, string.reverse consume memory
do
    print(runtime.callcontext({kill={memory=4000}}, string.lower, s))
    --> =done	hello

    print(runtime.callcontext({kill={memory=4000}}, string.lower, s1000))
    --> =killed


    -- string.upper consumes memory

    print(runtime.callcontext({kill={memory=4000}}, string.upper, s))
    --> =done	HELLO

    print(runtime.callcontext({kill={memory=4000}}, string.upper, s1000))
    --> =killed


    -- string.reverse consumes memory

    print(runtime.callcontext({kill={memory=4000}}, string.reverse, s))
    --> =done	OLleh

    print(runtime.callcontext({kill={memory=4000}}, string.reverse, s1000))
    --> =killed
end

-- string.sub consumes memory
do
    print(runtime.callcontext({kill={memory=1000}}, string.sub, s, 3, 2000))
    --> =done	lLO

    print(runtime.callcontext({kill={memory=1000}}, string.sub, s1000, 3, 2000))
    --> =killed
end

-- string.byte consumes memory
do
    -- helper function to consume the returned bytes from string.byte.
    function len(s)
        return select('#', s:byte(1, #s))
    end

    print(len("foobar"))
    --> =6

    print(runtime.callcontext({kill={memory=10000}}, len, s1000))
    --> =killed

    -- string.char consumes memory

    print(runtime.callcontext({kill={memory=1000}}, string.char, s:byte(1, #s)))
    --> =done	helLO

    print(runtime.callcontext({kill={memory=1000}}, string.char, s1000:byte(1, 1200)))
    --> =killed


    -- string.rep consumes memory

    print(runtime.callcontext({kill={memory=1000}}, string.rep, "ha", 10))
    --> =done	hahahahahahahahahaha

    print(runtime.callcontext({kill={memory=1000}}, string.rep, "ha", 600))
    --> =killed
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
    local ctx = runtime.callcontext({kill={cpu=1000}}, string.dump, mk(10))
    print(ctx)
    --> =done
    
    -- One with 500 lines runs out of CPU
    print(runtime.callcontext({kill={cpu=1000}}, string.dump, mk(500)))
    --> =killed

    -- A function with 12 lines is ok for mem
    local ctx = runtime.callcontext({kill={memory=1000}}, string.dump, mk(10))
    print(ctx)
    --> =done
    
    -- One with 500 lines runs out of mem
    print(runtime.callcontext({kill={cpu=1000}}, string.dump, mk(500)))
    --> =killed
end

-- string.format consumes memory and cpu
do
    -- a long format needs to be scanned and uses cpu
    print(runtime.callcontext({kill={cpu=1000}}, string.format, ("a"):rep(50)))
    --> =done	aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa

    print(runtime.callcontext({kill={cpu=1000}}, string.format, ("a"):rep(1000)))
    --> =killed

    -- format requires memory to build the formatted string
    local s = "aa"
    print(runtime.callcontext({kill={memory=1000}}, string.format, "%s %s %s %s %s", s, s, s, s, s))
    --> =done	aa aa aa aa aa

    s = ("a"):rep(1000)
    print(runtime.callcontext({kill={memory=5000}}, string.format, "%s %s %s %s %s", s, s, s, s, s))
    --> =killed
end
