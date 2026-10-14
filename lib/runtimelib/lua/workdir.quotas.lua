-- The 'cwd' field of a context sets the directory relative paths resolve
-- against.  Like the dirs in 'fs', a relative cwd is relative to the current
-- context's working directory.

local function readline(name)
    local f = assert(io.open(name))
    local line = f:read("l")
    f:close()
    return line
end

-- A sandbox whose working directory is the granted directory
print(runtime.callcontext({cwd="testfiles/sandbox", fs={{dir="testfiles/sandbox", allowed="r"}}}, function()
    print(readline("existing.txt"))
    --> =hello

    print(readline("./existing.txt"))
    --> =hello

    print(dofile("helper.lua"))
    --> =helper loaded

    print(loadfile("helper.lua")())
    --> =helper loaded

    -- require uses package.path, which is "./?.lua;./?/init.lua" by default
    print(require("helper"))
    --> ~helper loaded.*

    -- Files keep the name they were opened with
    local f = io.open("existing.txt")
    print(f)
    --> =file ("existing.txt")
    f:close()

    -- Errors show paths as written, not as resolved
    print(io.open("missing.txt"))
    --> ~nil\t.* missing.txt: .*

    -- Relative paths outside the grant are still not allowed
    print(pcall(io.open, "../other/secret.txt"))
    --> ~false\t.*safeio: operation not allowed

    -- A nested working directory is relative to this one; it can be outside
    -- the grant, but then relative paths there are not allowed.
    print(runtime.callcontext({cwd=".."}, function()
        print(readline("sandbox/existing.txt"))
        --> =hello

        print(pcall(io.open, "other/secret.txt"))
        --> ~false\t.*safeio: operation not allowed
    end))
    --> =done
end))
--> =done

-- Unrestricted contexts can have a working directory too
print(runtime.callcontext({cwd="testfiles/other"}, function()
    print(readline("secret.txt"))
    --> =secret
end))
--> =done

-- Outside the context, paths are relative to the process's working directory
print(readline("testfiles/sandbox/existing.txt"))
--> =hello

print(pcall(runtime.callcontext, {cwd=42}, print))
--> ~false\t.*cwd must be a non-empty string
