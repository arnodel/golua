local f = load("print('hello')")
f()
--> =hello

do
    local env = {x = 1}
    local f = load("x = 2", "chunk", "bt", env)
    print(env.x)
--> =1
    f()
    print(env.x)
--> =2
end

load("print(...)")(1, 2)
--> =1	2

-- This loads and executes the given file
loadfile("lua/loadfile.lua.notest")()
--> =loadfile

print(pcall(loadfile, "lua/nonexistent_file"))
--> ~false	loadfile: error reading file: .*

dofile("lua/loadfile.lua.notest")
--> =loadfile
