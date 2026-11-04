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

loadfile("lua/loadfile.lua.notest", "t", {print=print, ggg = "global"})()
--> =global

print(pcall(loadfile, "lua/loadfile.lua.notest", 123))
--> ~false\t.*must be a string

print(loadfile("lua/nonexistent_file"))
--> ~nil\t

dofile("lua/loadfile.lua.notest")
--> =loadfile

load(coroutine.wrap(function ()
    coroutine.yield("print(")
    coroutine.yield("'hello')")
end))()
--> =hello

load(coroutine.wrap(function ()
    coroutine.yield("print(")
    coroutine.yield("'abc')")
    coroutine.yield("")
    coroutine.yield("print(")
    coroutine.yield("'xyz')")
end))()
--> =abc

print(load(function() error("argh") end))
--> ~nil\t.*argh

print(load(function() return {} end))
--> ~nil\t.*must return a string

print(pcall(load, {}))
--> ~false\t.*string or function

print(load("\x00", "", "t"))
--> ~nil\t.*invalid token

print(load("hi", "", "b"))
--> ~nil\t.*text chunk

local z = "haha"
load(string.dump(function() print("hi", z) end))()
--> =hi	nil

-- A binary chunk of another format version is refused.  The version is the
-- byte after the 3-byte prefix.
do
    local d = string.dump(function() end)
    print(load(d:sub(1, 3) .. string.char(d:byte(4) + 1) .. d:sub(5)))
end
--> ~nil\tbinary chunk format [0-9]+, this golua reads format [0-9]+

print(load("???", "", "t"))
--> ~nil\t.*invalid token