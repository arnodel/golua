-- tags: !windows

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
    print(pcall(io.popen))
    --> ~^false\t.*value needed

    print(pcall(io.popen, {}))
    --> ~^false\t.*must be a string

    print(pcall(io.popen, "aaa", false))
    --> ~^false\t.*must be a string

    local fh = io.popen("hello")
    print(fh)
    --> =file ("hello")

    local f = io.popen("cat files/iotest.txt")

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

