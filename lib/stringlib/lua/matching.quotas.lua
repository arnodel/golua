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
    print(runtime.callcontext({memlimit=1000}, string.find, "abbbbbbbbbbc", "(b+)"))
    --> =done	2	11	bbbbbbbbbb

    print(runtime.callcontext({memlimit=3000}, string.find, "a"..("b"):rep(1000).."c", "(((b+)))"))
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
    print(runtime.callcontext({memlimit=1000}, string.match, "abbbbbbbbbbc", "(b+)"))
    --> =done	bbbbbbbbbb

    print(runtime.callcontext({memlimit=3000}, string.match, "a"..("b"):rep(1000).."c", "(((b+)))"))
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
    print(runtime.callcontext({memlimit=1000}, countwords, ("hello"):rep(10, " ")))
    --> =done
    print(wc)
    --> =10

    print(runtime.callcontext({memlimit=1000}, countwords, ("hello"):rep(1000, " ")))
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
    print(runtime.callcontext({memlimit=1000}, string.gsub, "1234567890", "%w", ("a"):rep(100)))
    --> =killed

    -- 2. Replacement function

    print(runtime.callcontext({memlimit=1000}, string.gsub, "1234567890", "%w", function(x) return x:rep(2) end))
    --> =done	11223344556677889900	10

    -- Building the substitution consumes memory
    print(runtime.callcontext({memlimit=1000}, string.gsub, "1234567890", "%w", function(x) return x:rep(100) end))
    --> =killed

    -- 3. Replacement table

    local t = {
        a = "A",
        b = ("B"):rep(100),
    }

    print(runtime.callcontext({memlimit=1000}, string.gsub, ("a"):rep(10), ".", t))
    --> =done	AAAAAAAAAA	10

    -- Building the substitution consumes memory
    print(runtime.callcontext({memlimit=1000}, string.gsub, ("b"):rep(100), ".", t))
    --> =killed
end
