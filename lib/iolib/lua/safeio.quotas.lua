-- When iosafe is on, functions opening files return errors

runtime.callcontext({flags="iosafe"}, function()

    -- io module funtions are unavailable

    print(pcall(io.open, "foo.txt"))
    --> ~false\t.*: safeio: operation not allowed

    print(pcall(io.input, "foo.txt"))
    --> ~false\t.*: safeio: operation not allowed

    print(pcall(io.lines, "foo.txt"))
    --> ~false\t.*: safeio: operation not allowed

    print(pcall(io.output, "foo.txt"))
    --> ~false\t.*: safeio: operation not allowed

    print(pcall(io.tmpfile))
    --> ~false\t.*: safeio: operation not allowed

end)

-- But functions operating on open files still work

local f = io.open("files/hello.txt")

runtime.callcontext({flags="iosafe"}, function()
    for line in f:lines() do
        print(line)
    end
    --> =bonjour
    f:seek("set")
    print(f:read(5))
    --> =bonjo
    f:close()
end)

local f = io.open("files/writetest-safeio.txt", "w")
runtime.callcontext({flags="iosafe"}, function()
    f:write("some text"):flush():close()
end)

print(io.open("files/writetest-safeio.txt", "r"):read("a"))
--> =some text
