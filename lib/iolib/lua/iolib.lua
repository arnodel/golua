local function testf(f)
    return function(...)
        (function(ok, ...) 
            if ok then
                print("OK", ...)
            else
                print("ERR", ...)
            end
        end)(pcall(f, ...))
    end
end

do
    local close = testf(io.close)
    close("abc")
    --> ~ERR	.*must be a file

    print(pcall(io.open))
    --> ~^false\t.*value needed

    print(pcall(io.open, {}))
    --> ~^false\t.*must be a string

    print(pcall(io.open, "aaa", false))
    --> ~^false\t.*must be a string

    testf(io.open)("files/doesnotexist")
    --> ~OK\tnil\t

    local f = io.open("files/iotest.txt")
    print(f)
    --> =file ("files/iotest.txt")

    print(pcall(f.read))
    --> ~^false\t.*value needed

    print(pcall(f.read, 123))
    --> ~^false\t

    print(pcall(f.read, f, "?"))
    --> ~^false\t.*invalid format

    testf(io.type)()
    --> ~ERR	.*value needed

    print(io.type(123))
    --> =nil

    print(io.type(f))
    --> =file

    print(pcall(f.lines))
    --> ~^false\t.*value needed

    print(pcall(f.lines, 123))
    --> ~^false\t.*must be a file

    print(pcall(f.lines, f, "wat"))
    --> ~^false\t.*invalid format


    for line in f:lines() do
        print(line)
    end
    --> =hello
    --> =123
    --> =bye

    testf(f.close)()
    --> ~ERR	.*value needed

    testf(f.flush)()
    --> ~ERR	.*value needed

    testf(f.flush)(123)
    --> ~ERR	.*must be a file

    f:close()
    print(io.type(f))
    --> =closed file
end

do
    print(pcall(io.popen))
    --> ~^false\t.*value needed

    print(pcall(io.popen, {}))
    --> ~^false\t.*must be a string

    print(pcall(io.popen, "aaa", false))
    --> ~^false\t.*must be a string

    local f = io.popen("cat files/iotest.txt")
    print(f)
    --> =file ("cat files/iotext.txt")

    print(pcall(f.read))
    --> ~^false\t.*value needed

    print(pcall(f.read, 123))
    --> ~^false\t

    print(pcall(f.read, f, "?"))
    --> ~^false\t.*invalid format

    testf(io.type)()
    --> ~ERR	.*value needed

    print(io.type(123))
    --> =nil

    print(io.type(f))
    --> =file

    print(pcall(f.lines))
    --> ~^false\t.*value needed

    print(pcall(f.lines, 123))
    --> ~^false\t.*must be a file

    print(pcall(f.lines, f, "wat"))
    --> ~^false\t.*invalid format


    for line in f:lines() do
        print(line)
    end
    --> =hello
    --> =123
    --> =bye

    testf(f.close)()
    --> ~ERR	.*value needed

    testf(f.flush)()
    --> ~ERR	.*value needed

    testf(f.flush)(123)
    --> ~ERR	.*must be a file

    f:close()
    print(io.type(f))
    --> =closed file
end

do
    testf(io.lines)(123)
    --> ~ERR	.*must be a string

    testf(io.lines)("nonexistent")
    --> ~ERR	.*

    testf(io.lines)("files/iotest.txt", "z")
    --> ~ERR	.*invalid format

    for line in io.lines("files/iotest.txt") do
        print(line)
    end
    --> =hello
    --> =123
    --> =bye

    print((pcall(io.lines, "files/missing")))
    --> =false
end

do
    local function wp(x)
        print("[" .. tostring(x) .. "]")
    end
    local f = io.open("files/writetest.txt", "w")

    testf(f.write)()
    --> ~ERR	.*value needed

    print(pcall(f.write, 123))
    --> ~^false\t.*must be a file

    print(pcall(f.write, f, {}))
    --> ~^false\t.*must be a string or a number

    f:write("foobar", 1234, "\nabc\n")
    f:close()
    f = io.open("files/writetest.txt", "r")

    wp(f:read("a"))
    --> =[foobar1234
    --> =abc
    --> =]

    wp(f:read("l"))
    --> =[nil]

    testf(f.seek)(f, "set", "hello")
    --> ~ERR	.*must be an integer

    f:seek("set", 0)
    wp(f:read(7))
    --> =[foobar1]

    f:seek("cur", 3)
    wp(f:read(10))
    --> =[
    --> =abc
    --> =]

    f:seek("end", -4)
    wp(f:read("L"))
    --> =[abc
    --> =]

    f:seek("end", -2)
    f:seek("cur", -2)
    wp(f:read("L"))
    --> =[abc
    --> =]
    
    print(pcall(f.seek))
    --> ~false\t.*value needed

    print((pcall(f.seek, "hello")))
    --> =false

    print(pcall(f.seek, f, 42))
    --> ~^false\t.*string

    print(f:seek("set", -100000))
    --> ~^nil\t

    local metaf = getmetatable(f)

    testf(metaf.__tostring)()
    --> ~ERR	.*value needed

    testf(metaf.__tostring)("not a file")
    --> ~ERR	.*must be a file
end

do
    local function wp(x)
        print("[" .. tostring(x) .. "]")
    end
    local f = io.popen("cat > files/popenwrite.txt", "w")

    testf(f.write)()
    --> ~ERR	.*value needed

    print(pcall(f.write, 123))
    --> ~^false\t.*must be a file

    print(pcall(f.write, f, {}))
    --> ~^false\t.*must be a string or a number

    f:write("foobar", 1234, "\nabc\n")
    f:close()
    f = io.open("files/popenwrite.txt", "r")

    wp(f:read("a"))
    --> =[foobar1234
    --> =abc
    --> =]

    wp(f:read("l"))
    --> =[nil]
end

do
    testf(io.read)("z")
    --> ~ERR	.*invalid format

    testf(io.read)(false)
    --> ~ERR	.*invalid format

    local f = io.open("files/writetest2.txt", "w+")
    io.output(f)
    io.write([[Dear sir,
Blah blah,

Yours sincerely.
]])
    io.flush()
    io.input(f)
    f:seek("set", 0)
    print(io.read())
    --> =Dear sir,

    for line in io.lines() do
        print(line)
    end
    --> =Blah blah,
    --> =
    --> =Yours sincerely.
end

do
    local f = io.tmpfile()
    print(io.type(f))
    --> =file

    print(pcall(f.seek, f, "wat"))
    --> ~false\t.*: #1 must be "cur", "set" or "end"

    -- TODO: do something with the file
end

do
    -- local stdin = io.input()
    -- print(io.type(stdin))
    -- --> =file

    io.input("files/iotest.txt")
    print(io.input())
    --> =file ("files/iotest.txt")

    print((pcall(io.input, "files/missing")))
    --> =false

    print((pcall(io.input, 123)))
    --> =false
end

do
    -- local stdout = io.output()
    -- print(io.type(stdout))
    -- --> =file

    testf(io.output)(false)
    --> ~^ERR

    io.output("files/outputtest.txt")
    io.write("hello")
    io.write("bye")
    io.close()
    print(pcall(io.close))
    --> ~false\t.*file already closed
    print(io.open("files/outputtest.txt"):read())
    --> =hellobye
end

do
    local f = io.open("files/iotest.txt")
    print(f:read(0))
    --> =
    print(f:read("L"))
    --> =hello
    --> =
    print(f:read("n"))
    --> =123
    print(f:read("n"))
    --> =nil
    print(f:read("a"))
    --> =bye
    --> =
    print(f:read(0))
    --> =nil
end

do
    print(io.stdout:close())
    --> ~nil\t.*cannot close standard file

    print(io.stdin:close())
    --> ~nil\t.*cannot close standard file

    print(io.stderr:close())
    --> ~nil\t.*cannot close standard file
end

local ff -- this is to remember the file f below refers to after f goes out of scope.
do
    local f <close> = io.open("files/writetest3.txt", "w")
    ff = f
    -- hard to test what it does, at least make sure the correct modes are accepted
    f:setvbuf("no")
    f:setvbuf("full")
    f:setvbuf("line")
    f:setvbuf("full", 0)
    f:setvbuf("full", 2000)
    f:setvbuf("line", 1000)

    local function perr(...)
        print(pcall(function(...) f:setvbuf(...) end, ...))
    end

    perr("line", -1)
    --> ~false\t.*invalid buffer size

    perr("full", -1000)
    --> ~false\t.*invalid buffer size

    perr("blah")
    --> ~false\t.*invalid buffer mode

    perr()
    --> ~false\t.*2 arguments needed

    perr(100)
    --> ~false\t.*#2 must be a string

    perr("full", {})
    --> ~false\t.*#3 must be an integer

    -- ff is still open at this point
    print(io.type(ff))
    --> =file
end

-- New in Lua 5.4:
-- the __close metamethod was called, so ff should be closed now.
print(io.type(ff))
--> =closed file

do
    local close = getmetatable(ff).__close

    print(pcall(close))
    --> ~false\t.*value needed

    print(pcall(close, {}))
    --> ~false\t.*#1 must be a file
end
