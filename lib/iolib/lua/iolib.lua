do
    local f = io.open("files/iotest.txt")
    print(io.type(f))
    --> =file

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
    local function wp(x)
        print("[" .. x .. "]")
    end
    local f = io.open("files/writetest.txt", "w")
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
