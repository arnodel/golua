-- utf8.char
do
    -- utf8.char uses memory to build a string
    local ctx, s = runtime.callcontext({memlimit=10000}, utf8.char, utf8.codepoint(("x"):rep(100), 1, 100))
    print(ctx, #s)
    --> =done	100

    print(runtime.callcontext({memlimit=10000}, utf8.char, utf8.codepoint(("x"):rep(1000), 1, 1000)))
    --> =killed

    -- utf8.cahr uses cpu to build a string
    local ctx, s = runtime.callcontext({kill={cpu=1000}}, utf8.char, utf8.codepoint(("x"):rep(100), 1, 100))
    print(ctx, #s)
    --> =done	100

    print(runtime.callcontext({kill={cpu=1000}}, utf8.char, utf8.codepoint(("x"):rep(1000), 1, 1000)))
    --> =killed
end


-- utf8.codes
do
    local function len(s)
        local len = 0
        for x in utf8.codes(s) do
            len = len + 1
        end
        return len
    end

    -- Iterating over utf8.codes(s) consumes cpu
    print(runtime.callcontext({kill={cpu=1000}}, len, ("s"):rep(50)))
    --> =done	50

    print(runtime.callcontext({kill={cpu=1000}}, len, ("s"):rep(500)))
    --> =killed

    -- It doesn't consume memory
    ctx1 = runtime.callcontext({memlimit=1000}, len, ("s"):rep(100))
    ctx2 = runtime.callcontext({memlimit=1000}, len, ("s"):rep(1000))
    print(ctx1, ctx2, ctx2.used.memory / ctx1.used.memory < 1.2)
    --> =done	done	true
end

-- utf8.codepoint
do
    local function codepoint(s)
        -- drop the output to prevent requiring memory for it
        utf8.codepoint(s, 1, #s)
    end

    -- utf8.codepoint requires cpu proportional to input size

    print(runtime.callcontext({kill={cpu=1000}}, codepoint, ("a"):rep(500)))
    --> =done

    print(runtime.callcontext({kill={cpu=1000}}, codepoint, ("a"):rep(1000)))
    --> =killed
end

-- utf8.len
do
    -- utf8.len requires cpu proportional to input size

    print(runtime.callcontext({kill={cpu=1000}}, utf8.len, ("a"):rep(500)))
    --> =done	500

    print(runtime.callcontext({kill={cpu=1000}}, utf8.len, ("a"):rep(1000)))
    --> =killed
end

-- utf8.offset
do
    -- utf8.offset requires cpu proportional to the displacement

    print(runtime.callcontext({kill={cpu=1000}}, utf8.offset, ("日本誒"):rep(100), 200))
    --> =done	598

    print(runtime.callcontext({kill={cpu=1000}}, utf8.offset, ("日本誒"):rep(100), -200))
    --> =done	301

    print(runtime.callcontext({kill={cpu=2000}}, utf8.offset, ("日本誒"):rep(1000), 2000))
    --> =killed

    print(runtime.callcontext({kill={cpu=2000}}, utf8.offset, ("日本誒"):rep(1000), -2000))
    --> =killed
end

