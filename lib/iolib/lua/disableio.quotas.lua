-- When IO is disabled, most functions return errors

runtime.callcontext({flags="iosafe"}, function()

    -- io module funtions are unavailable

    print(pcall(io.close, 1))
    --> =false	missing flags: iosafe

    print(pcall(io.flush, 1))
    --> =false	missing flags: iosafe

    print(pcall(io.input, 1))
    --> =false	missing flags: iosafe

    print(pcall(io.lines, "foo"))
    --> =false	missing flags: iosafe

    print(pcall(io.open, "foo"))
    --> =false	missing flags: iosafe

    print(pcall(io.output, 1))
    --> =false	missing flags: iosafe

    print(pcall(io.read, "foo"))
    --> =false	missing flags: iosafe
   
    print(pcall(io.tmpfile, "foo"))
    --> =false	missing flags: iosafe

    print(pcall(io.write, "foo"))
    --> =false	missing flags: iosafe

    -- functions on files are unavailable

    print(pcall(io.stdin.read, stdin))
    --> =false	missing flags: iosafe

    print(pcall(io.stdin.lines, stdin))
    --> =false	missing flags: iosafe

    print(pcall(io.stdout.close, stdout))
    --> =false	missing flags: iosafe

    print(pcall(io.stdout.flush, stdout))
    --> =false	missing flags: iosafe

    print(pcall(io.stdin.seek, stdin, 2, 3))
    --> =false	missing flags: iosafe

    print(pcall(io.stdin.write, stdout))
    --> =false	missing flags: iosafe

end)
