-- When IO is disabled, most functions return errors

runtime.callcontext({io="off"}, function()

    -- io module funtions are unavailable

    print(pcall(io.close, 1))
    --> =false	io disabled

    print(pcall(io.flush, 1))
    --> =false	io disabled

    print(pcall(io.input, 1))
    --> =false	io disabled

    print(pcall(io.lines, "foo"))
    --> =false	io disabled

    print(pcall(io.open, "foo"))
    --> =false	io disabled

    print(pcall(io.output, 1))
    --> =false	io disabled

    print(pcall(io.read, "foo"))
    --> =false	io disabled
   
    print(pcall(io.tmpfile, "foo"))
    --> =false	io disabled

    print(pcall(io.write, "foo"))
    --> =false	io disabled

    -- functions on files are unavailable

    print(pcall(io.stdin.read, stdin))
    --> =false	io disabled

    print(pcall(io.stdin.lines, stdin))
    --> =false	io disabled

    print(pcall(io.stdout.close, stdout))
    --> =false	io disabled

    print(pcall(io.stdout.flush, stdout))
    --> =false	io disabled

    print(pcall(io.stdin.seek, stdin, 2, 3))
    --> =false	io disabled

    print(pcall(io.stdin.write, stdout))
    --> =false	io disabled
end)
