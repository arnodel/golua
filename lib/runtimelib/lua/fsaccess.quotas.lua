-- The 'fs' field of a context is a list of grants of the form
--   {dir="path/to/dir", allowed=ACTIONS}
-- where ACTIONS is a string made of the following characters:
--   'r': read a file
--   'w': write to an existing file
--   'c': create a file
--   'd': delete or rename a file
--   '*': all of the above
-- Files outside the granted directories cannot be accessed at all.

local function readline(name)
    local f = assert(io.open(name))
    local line = f:read("l")
    f:close()
    return line
end

-- Read-only access to testfiles/sandbox
print(runtime.callcontext({fs={{dir="testfiles/sandbox", allowed="r"}}}, function()
    print(readline("testfiles/sandbox/existing.txt"))
    --> =hello

    -- Paths are normalised before checking
    print(readline("./testfiles/sandbox/../sandbox/existing.txt"))
    --> =hello

    -- Files outside the directory cannot be accessed
    print(pcall(io.open, "testfiles/other/secret.txt"))
    --> ~false\t.*safeio: operation not allowed

    print(pcall(io.open, "testfiles/sandbox/../other/secret.txt"))
    --> ~false\t.*safeio: operation not allowed

    -- Writing is not allowed
    print(pcall(io.open, "testfiles/sandbox/existing.txt", "a"))
    --> ~false\t.*safeio: operation not allowed

    -- Neither are functions that cannot be restricted
    print(pcall(io.popen, "ls"))
    --> ~false\t.*missing flags: iosafe
end))
--> =done

-- Write but not create
print(runtime.callcontext({fs={{dir="testfiles/sandbox", allowed="rw"}}}, function()
    local f = assert(io.open("testfiles/sandbox/existing.txt", "a"))
    f:close()

    print(pcall(io.open, "testfiles/sandbox/new.txt", "w"))
    --> ~false\t.*safeio: operation not allowed
end))
--> =done

-- All actions
print(runtime.callcontext({fs={{dir="testfiles/sandbox", allowed="*"}}}, function()
    local f = assert(io.open("testfiles/sandbox/new.txt", "w"))
    f:write("new\n")
    f:close()

    print(os.rename("testfiles/sandbox/new.txt", "testfiles/sandbox/renamed.txt"))
    --> =true

    -- Files cannot be moved out of the directory
    print(pcall(os.rename, "testfiles/sandbox/renamed.txt", "testfiles/other/moved.txt"))
    --> ~false\t.*safeio: operation not allowed

    print(readline("testfiles/sandbox/renamed.txt"))
    --> =new

    print(os.remove("testfiles/sandbox/renamed.txt"))
    --> =true

    -- Temporary files are created in the system temp dir, which is not granted
    print(pcall(io.tmpfile))
    --> ~false\t.*safeio: operation not allowed
end))
--> =done

-- Nested contexts can only narrow access
print(runtime.callcontext({fs={{dir="testfiles", allowed="r"}}}, function()
    print(runtime.callcontext({fs={{dir="testfiles/sandbox", allowed="*"}}}, function()
        print(readline("testfiles/sandbox/existing.txt"))
        --> =hello

        -- Only reading was granted by the parent context
        print(pcall(io.open, "testfiles/sandbox/new.txt", "w"))
        --> ~false\t.*safeio: operation not allowed

        print(pcall(io.open, "testfiles/other/secret.txt"))
        --> ~false\t.*safeio: operation not allowed
    end))
    --> =done

    -- A directory not granted by the parent cannot be granted
    print(pcall(runtime.callcontext, {fs={{dir="..", allowed="r"}}}, print))
    --> ~false\t.*cannot grant access to "..": safeio: operation not allowed
end))
--> =done

-- An empty list of grants means no file access
print(runtime.callcontext({fs={}}, pcall, io.open, "testfiles/sandbox/existing.txt"))
--> ~done\tfalse\t.*safeio: operation not allowed

-- Invalid grants
print(pcall(runtime.callcontext, {fs={{allowed="r"}}}, print))
--> ~false\t.*fs grant #1: dir must be a non-empty string

print(pcall(runtime.callcontext, {fs={{dir="testfiles", allowed="rx"}}}, print))
--> ~false\t.*fs grant #1: invalid file action 'x'

print(pcall(runtime.callcontext, {fs={{dir="testfiles/nonexistent", allowed="r"}}}, print))
--> ~false\t.*cannot grant access to "testfiles/nonexistent"
