local function errtest(f)
    return function(...)
        local ok, err = pcall(f, ...)
        if ok then
            print"OK"
        else
            print(err)
        end
    end
end

do
    local function pf(...)
        print(string.find(...))
    end
    
    pf("a", "a", 1, true)
    --> =1	1

    pf("hello world!", "o w", 1, true)
    --> =5	7

    pf("xyzt", "yt", 1, true)
    --> =nil

    pf("1234 abc453", "%l+")
    --> =6	8

    pf("  foo=[a [lovely] day];", "(%w+)=(%b[])")
    --> =3	22	foo	[a [lovely] day]

    print(pcall(string.find, "abc", "(xx%1)"))
    --> ~false\t.*

    local errf = errtest(string.find)

    errf(1)
    --> ~2 arguments needed

    errf("x", "y", "z")
    --> ~must be an integer

    pf("x", "x", 3)
    --> =nil

    pf("x", "x", -10)
    --> =nil

    pf("x", "y")
    --> =nil
end

do
    local function pm(...)
        print(string.match(...))
    end

    pm("Let me *stress* that I *am*", "%*.-%*")
    --> =*stress*

    pm("Let me *stress* that I *am*", "%*.-%*", 17)
    --> =*am*

    pm("Let me *stress* that I *am*", "%*(.-)%*")
    --> =stress

    pm("Let me *stress* that I *am*", "%*(.-)%*", 17)
    --> =am

    pm("A *bold* and an _underline_", "([*~_])(.-)%1")
    --> =*	bold

    pm("A *b_o_l_d* and an _under~line_", "([*~_])(.-)%1")
    --> =*	b_o_l_d

    pm("A *b_o_l_d* and an _under~line_", "([*~_])(.-)%1", 10)
    --> =_	under~line

    local errm = errtest(string.match)

    errm("x")
    --> ~2 arguments needed

    errm("x", "%")
    --> ~malformed pattern
end

do
    local s = "hello world from Lua"
    for w in string.gmatch(s, "%a+") do
        print(w)
    end
    --> =hello
    --> =world
    --> =from
    --> =Lua

    local t = {}
    local s = "from=world, to=Lua"
    for k, v in string.gmatch(s, "(%w+)=(%w+)") do
        t[k] = v
    end
    print(t.from, t.to)
    --> =world	Lua

    local errgm = errtest(string.gmatch)

    errgm()
    --> ~2 arguments needed

    errgm("x")
    --> ~2 arguments needed

    errgm("x", "%")
    --> ~malformed pattern

    for w in string.gmatch("abc", "b*") do
        print(w)
    end
    --> =
    --> =b
    --> =
end

do
    local function pgs(...)
        print(string.gsub(...))
    end

    pgs("hello world", "(%w+)", "%1 %1")
    --> =hello hello world world	2

    pgs("hello world", "%w+", "%0 %0", 1)
    --> =hello hello world	1

    pgs("hello world from Lua", "(%w+)%s*(%w+)", "%2 %1")
    --> =world hello Lua from	2

    local function getenv(v)
        if v == "HOME" then
            return "/home/roberto"
        elseif v == "USER" then
            return "roberto"
        end
    end

    pgs("home = $HOME, user = $USER", "%$(%w+)", getenv)
    --> =home = /home/roberto, user = roberto	2

    pgs("4+5 = $return 4+5$", "%$(.-)%$",
                      function (s)
                          return load(s)()
                      end
    )
    --> =4+5 = 9	1

    local t = {name="lua", version="5.3"}
    pgs("$name-$version.tar.gz", "%$(%w+)", t)
    --> =lua-5.3.tar.gz	2

    local errgs = errtest(string.gsub)

    errgs()
    --> ~3 arguments needed

    errgs("x", "y")
    --> ~3 arguments needed

    errgs("x", "y", "z", "t")
    --> ~must be an integer

    errgs("x", "%", "z")
    --> ~malformed pattern

    pgs("xyz", "()y()", "%1-%2")
    --> =x2-3z	1

    errgs("xyz", "(x)", "%2")
    --> ~invalid capture index

    pgs("xyz", "xyz", "%%")
    --> =%	1

    errgs("xyz", "xyz", "%x %%")
    --> ~invalid.*%

    local replt = {}
    setmetatable(replt, {__index=function() error("boo") end})
    errgs("xyz", "xyz", replt)
    --> ~boo

    errgs("xyz", "xyz", function () error("baa") end)
    --> ~baa

    errgs("xyz", "xyz", false)
    --> ~must be a string, table or function
    
    errgs("z", "z", {z=true})
    --> ~invalid replacement

    pgs("z", "z", {z=false})
    --> =z	1

    pgs("abc", "b*", "Z")
    --> =ZaZcZ	4
end
