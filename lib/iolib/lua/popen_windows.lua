-- tags: windows

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

    local f = io.popen("type files/iotest.txt")

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

