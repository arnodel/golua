-- Tests for string.find
do
    -- string.find in plain mode consumes cpu proportional to search string
    print(runtime.callcontext({kill={cpu=1000}}, string.find, ("straw"):rep(20).."needle", "needle", 1, true))
    --> =done	101	106

    print(runtime.callcontext({kill={cpu=1000}}, string.find, ("straw"):rep(200).."needle", "needle", 1, true))
    --> =killed

    -- string.find in pattern mode consumes cpu proportional to the amount of
    -- searching
    print(runtime.callcontext({kill={cpu=10000}}, string.find, ("a"):rep(50), ".-b"))
    --> =done	nil

    print(runtime.callcontext({kill={cpu=10000}}, string.find, ("a"):rep(500), ".-b"))
    --> =killed

    -- captures consumes memory
    print(runtime.callcontext({kill={memory=1000}}, string.find, "abbbbbbbbbbc", "(b+)"))
    --> =done	2	11	bbbbbbbbbb

    print(runtime.callcontext({kill={memory=3000}}, string.find, "a"..("b"):rep(1000).."c", "(((b+)))"))
    --> =killed
end

-- Tests for string.match
do
    -- string.match consumes cpu proportional to the amount of searching
    print(runtime.callcontext({kill={cpu=10000}}, string.match, ("a"):rep(50), ".-b"))
    --> =done	nil

    print(runtime.callcontext({kill={cpu=10000}}, string.match, ("a"):rep(500), ".-b"))
    --> =killed

    -- captures consumes memory
    print(runtime.callcontext({kill={memory=1000}}, string.match, "abbbbbbbbbbc", "(b+)"))
    --> =done	bbbbbbbbbb

    print(runtime.callcontext({kill={memory=3000}}, string.match, "a"..("b"):rep(1000).."c", "(((b+)))"))
    --> =killed
end

-- Tests for string.gmatch
do
    local wc
    local function countwords(s)
        wc = 0
        for w in string.gmatch(s, "%w+") do
            wc = wc + 1
        end
    end

    -- every match returned consumes cpu
    print(runtime.callcontext({kill={cpu=1000}}, countwords, ("hello"):rep(10, " ")))
    --> =done
    print(wc)
    --> =10

    print(runtime.callcontext({kill={cpu=1000}}, countwords, ("hello"):rep(1000, " ")))
    --> =killed
    print(wc > 10 and wc < 200)
    --> =true

    -- every match returned consumes memory
    print(runtime.callcontext({kill={memory=1000}}, countwords, ("hello"):rep(10, " ")))
    --> =done
    print(wc)
    --> =10

    print(runtime.callcontext({kill={memory=1000}}, countwords, ("hello"):rep(1000, " ")))
    --> =killed
    print(wc > 10 and wc < 200)
    --> =true
end

-- Tests for string.gsub
do
    -- 1. Replacemement string

    print(runtime.callcontext({kill={cpu=1000}}, string.gsub, "a b c", "%w+", "%0 %0 %0"))
    --> =done	a a a b b b c c c	3

    -- It takes cpu to parse the input string
    print(runtime.callcontext({kill={cpu=1000}}, string.gsub, ("a"):rep(1000), "%w", "%0"))
    --> =killed

    -- It takes cpu to parse the replacement string
    print(runtime.callcontext({kill={cpu=1000}}, string.gsub, "a b c", "%w+", ("a"):rep(1000)))
    --> =killed

    -- Building the substitution consumes memory
    print(runtime.callcontext({kill={memory=1000}}, string.gsub, "1234567890", "%w", ("a"):rep(100)))
    --> =killed

    -- 2. Replacement function

    print(runtime.callcontext({kill={memory=1000}}, string.gsub, "1234567890", "%w", function(x) return x:rep(2) end))
    --> =done	11223344556677889900	10

    -- Building the substitution consumes memory
    print(runtime.callcontext({kill={memory=1000}}, string.gsub, "1234567890", "%w", function(x) return x:rep(100) end))
    --> =killed

    -- 3. Replacement table

    local t = {
        a = "A",
        b = ("B"):rep(100),
    }

    print(runtime.callcontext({kill={memory=1000}}, string.gsub, ("a"):rep(10), ".", t))
    --> =done	AAAAAAAAAA	10

    -- Building the substitution consumes memory
    print(runtime.callcontext({kill={memory=1000}}, string.gsub, ("b"):rep(100), ".", t))
    --> =killed
end

-- A match that runs out of CPU in a nested context counts against the
-- contexts around it
do
    -- This pattern takes a number of steps exponential in the length of s
    local s = ("a"):rep(50)
    local ptn = ("a?"):rep(50) .. s

    -- The nested context has what is left of the CPU of the outer one, so
    -- the outer one does not outlive it
    local n = 0
    print(runtime.callcontext({kill={cpu=100000}}, function()
        for i = 1, 10 do
            n = n + 1
            runtime.callcontext({}, string.find, s, ptn)
        end
    end))
    --> =killed
    print(n)
    --> =1

    -- The same when the match is in a protected call
    n = 0
    print(runtime.callcontext({kill={cpu=100000}}, function()
        for i = 1, 10 do
            n = n + 1
            pcall(string.find, s, ptn)
        end
    end))
    --> =killed
    print(n)
    --> =1

    -- A nested context with a limit of its own costs the outer one that much
    n = 0
    print(runtime.callcontext({kill={cpu=100000}}, function()
        for i = 1, 10 do
            n = n + 1
            runtime.callcontext({kill={cpu=30000}}, string.find, s, ptn)
        end
    end))
    --> =killed
    print(n)
    --> =4
end

-- A match is stopped by a time limit while it runs
do
    -- This pattern never matches s, and finding that out takes a number of
    -- steps exponential in the length of s, far more than there is time for
    local s = ("a"):rep(50)
    local ptn = ("a?"):rep(50) .. s .. "b"

    local ctx = runtime.callcontext({kill={millis=20}}, string.find, s, ptn)
    print(ctx)
    --> =killed

    -- It was stopped long before the end of the match
    print(ctx.used.cpu < 2^40)
    --> =true

    -- It didn't last much more than the limit (the bound is generous because
    -- a loaded machine can deschedule the process across the deadline)
    print(ctx.used.millis < 1000)
    --> =true
end
