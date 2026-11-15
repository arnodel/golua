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

-- The name of a chunk can start with a mark that says what the name is; the
-- mark is not shown in messages.  "=" is for a name to show as it is, "@" for
-- the name of a file.
do
    print(pcall(load("error('x')", "=name")))
    --> =false	name:1: x
    print(pcall(load("error('x')", "@dir/file.lua")))
    --> =false	dir/file.lua:1: x
    print(pcall(load("error('x')", "==name")))
    --> =false	=name:1: x
    print(pcall(load("error('x')", "=")))
    --> =false	:1: x
    print(pcall(load("local t; return t.x", "=name")))
    --> ~^false\tname:1: .*index

    -- A syntax error names the chunk the same way.
    print(select(2, load("x x", "=name")):match("^[^:]*"))
    --> =name
    print(select(2, load("x x", "@dir/file.lua")):match("^[^:]*"))
    --> =dir/file.lua

    -- A long name is shown as 59 bytes: the start of a "=" name, the end of
    -- a file name after "...".
    local function shown(name)
        local _, msg = pcall(load("error('x')", name))
        local _, syntaxmsg = load("x x", name)
        local s = msg:match("^(.-):1: x$")
        assert(syntaxmsg:match("^[^:]*") == s)
        return #s, s:sub(1, 5), s:sub(-5)
    end
    print(shown("=" .. string.rep("a", 56) .. "xyz"))
    --> =59	aaaaa	aaxyz
    print(shown("=" .. string.rep("a", 57) .. "xyz"))
    --> =59	aaaaa	aaaxy
    print(shown("=" .. string.rep("a", 200) .. "xyz"))
    --> =59	aaaaa	aaaaa
    print(shown("@" .. string.rep("a", 56) .. "xyz"))
    --> =59	aaaaa	aaxyz
    print(shown("@" .. string.rep("a", 57) .. "xyz"))
    --> =59	...aa	aaxyz
    print(shown("@" .. string.rep("a", 200) .. "xyz"))
    --> =59	...aa	aaxyz

    -- A name with no mark is cut to its first 59 bytes.
    print(shown(string.rep("a", 200)))
    --> =59	aaaaa	aaaaa
end

-- This loads and executes the given file
loadfile("lua/loadfile.lua.notest")()
--> =loadfile

loadfile("lua/loadfile.lua.notest", "t", {print=print, ggg = "global"})()
--> =global

print(pcall(loadfile, "lua/loadfile.lua.notest", {}))
--> ~false\t.*must be a string

print(loadfile("lua/nonexistent_file"))
--> ~nil\t

print(loadfile(1))
--> ~nil\topen 1:

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