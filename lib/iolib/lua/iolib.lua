do
    print(pcall(io.open))
    --> ~^false\t.*value needed

    print(pcall(io.open, {}))
    --> ~^false\t.*must be a string

    print(pcall(io.open, "aaa", false))
    --> ~^false\t.*must be a string

    local f = io.open("files/iotest.txt")
    print(f)
    --> =file("files/iotest.txt")

    print(pcall(f.read))
    --> ~^false\t.*value needed

    print(pcall(f.read, 123))
    --> ~^false\t

    print(pcall(f.read, f, "?"))
    --> ~^false\t.*invalid format

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

    f:close()
    print(io.type(f))
    --> =closed file
end

do
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
        print("[" .. x .. "]")
    end
    local f = io.open("files/writetest.txt", "w")

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

    print(pcall(f.seek))
    --> ~false\t.*value needed

    print((pcall(f.seek, "hello")))
    --> =false

    print(pcall(f.seek, f, 42))
    --> ~^false\t.*string

end

do
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
    --> =false	#1 must be "cur", "set" or "end"

    -- TODO: do something with the file
end

do
    local stdin = io.input()
    print(io.type(stdin))
    --> =file

    io.input("files/iotest.txt")
    print(io.input())
    --> =file("files/iotest.txt")

    print((pcall(io.input, "files/missing")))
    --> =false

    print((pcall(io.input, 123)))
    --> =false
end

do
    local stdout = io.output()
    print(io.type(stdout))
    --> =file

    io.output("files/outputtest.txt")
    io.write("hello")
    io.write("bye")
    io.close()
    print(io.open("files/outputtest.txt"):read())
    --> =hellobye
end
