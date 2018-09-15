do
    local s = "hello"
    print(string.byte(s))
    --> =104

    print(string.byte(s, -1))
    --> =111

    print(string.byte(s, 2, 4))
    --> =101	108	108

    print(string.byte(s, -5, 2))
    --> =104	101
    
    print(string.byte(s, -2, -1))
    --> =108	111

    -- Byte can also be called as a method on strings
    print(s:byte(3))
    --> =108
end

do
    print(string.char(65, 66, 67))
    --> =ABC

    print(pcall(string.char, -1))
    --> ~^false\t.*out of range.*

    print(pcall(string.char, 256))
    --> ~^false\t.*out of range.*
end

do
    print(string.len("abc"), string.len(""))
    --> =3	0
end

do
    local s = "ABCdef123"
    print(s:lower())
    --> =abcdef123

    print(s:upper())
    --> =ABCDEF123
end

do
    local s = "xy"
    for i = 0, 3 do
        print(s:rep(i))
    end
    --> =
    --> =xy
    --> =xyxy
    --> =xyxyxy

    for i = 0, 3 do
        print(s:rep(i, "--"))
    end
    --> =
    --> =xy
    --> =xy--xy
    --> =xy--xy--xy
end

do
    local s = "EGASSEM TERCES"
    print(s:reverse())
    --> =SECRET MESSAGE

    print(string.reverse("12345"))
    --> =54321
end

do
    local s = "abc"
    for i = -4, 4 do
        print(s:sub(i))
    end
    --> =abc
    --> =abc
    --> =bc
    --> =c
    --> =abc
    --> =abc
    --> =bc
    --> =c
    --> =
end

do
    local function pf(...)
        print(string.format(...))
    end
    
    pf("%s=%f", "pi", 3.14)
    --> =pi=3.140000

    pf("-%s-%s-%s", nil, true, false)
    --> =-nil-true-false

    pf("%% %q %%", [["hello"	123]])
    --> =% "\"hello\"\t123" %

    pf("%d//%5d//%-5d//%+d//%05d", 10.0, "10", 10, 10, 10)
    --> =10//   10//10   //+10//00010

    pf("%.2f~~%5.2f~~%-5.2f~~%+.2f~~%05.2f", 3.14, "3.14", 3.14, 3.14, 3.14)
    --> =3.14~~ 3.14~~3.14 ~~+3.14~~03.14

    -- To many values is OK
    pf("%s", 1, 2, 3)
    --> =1

    -- Not enough values is not OK
    print(pcall(pf, "%s %d", 1))
    --> ~^false\t.*$
    
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

end

do
    print(string.gsub("hello world", "(%w+)", "%1 %1"))
    --> =hello hello world world	2

    print(string.gsub("hello world", "%w+", "%0 %0", 1))
    --> =hello hello world	1

    print(string.gsub("hello world from Lua", "(%w+)%s*(%w+)", "%2 %1"))
    --> =world hello Lua from	2

    local function getenv(v)
        if v == "HOME" then
            return "/home/roberto"
        elseif v == "USER" then
            return "roberto"
        end
    end

    print(string.gsub("home = $HOME, user = $USER", "%$(%w+)", getenv))
    --> =home = /home/roberto, user = roberto	2

    print(string.gsub("4+5 = $return 4+5$", "%$(.-)%$",
                      function (s)
                          return load(s)()
                      end
    ))
    --> =4+5 = 9	1

    local t = {name="lua", version="5.3"}
    print(string.gsub("$name-$version.tar.gz", "%$(%w+)", t))
    --> =lua-5.3.tar.gz	2
end

do
    local function pack(...)
        print(string.byte(string.pack(...), 1, -1))
    end
    
    pack("bB", 100, 200)
    --> =100	200

    pack("<i", 1)
    --> =1	0	0	0

    pack(">i", 1)
    --> =0	0	0	1

    pack("!2bh", 10, 20)
    --> =10	0	20	0

    pack("<!4zs", "A", "BCD")
    --> =65	0	0	0	3	0	0	0	0	0	0	0	66	67	68

    pack("<i6", 123456789)
    --> =21	205	91	7	0	0

    pack(">i6", 123456789)
    --> =0	0	7	91	205	21

    pack(">i10", 9876543210)
    --> =0	0	0	0	0	2	76	176	22	234

    pack("<i10", 9876543210)
    --> =234	22	176	76	2	0	0	0	0	0

    pack("xx")
    --> =0	0

    pack("x!4Xjz", "A")
    --> =0	0	0	0	65	0

    pack("Hc4", 65535, "AB")
    --> =255	255	65	66	0	0

    pack("<d", 123.456)
    --> =119	190	159	26	47	221	94	64

    pack("<f", 1e-3)
    --> =111	18	131	58

    local function packError(...)
        if pcall(string.pack, ...) then
            print("NO ERROR")
        end
    end

    packError("d") -- missing value
    packError("i", "abc") -- bad value type
    packError("!17") -- size out of bounds
    packError("b", 128) -- value out of bounds
    packError("B", -1) -- value out of bounds
    packError("cxx", "abc") -- missing string length
    packError("c2", "abc") -- string too long
    packError("dy", 1) -- invalid option "y"
    packError("bbX", 1, 1) -- "X" must be followed by option
    packError("!3bi", 1, 1) -- alignment not a power of 2
end

do
    local function unpack(...)
        print(string.unpack(...))
    end

    unpack("b", "A")
    --> =65	2

    unpack("<i", "abcd")
    --> =1684234849	5

    unpack(">i", "abcd")
    --> =1633837924	5

    unpack("<bc2H", "Bhi\x00\x04")
    --> =66	hi	1024	6

    unpack(">s1xB", "\x05hello*\x80")
    --> =hello	128	9

    unpack("B", "1234\xff678", 5)
    --> =255	6

    unpack("<!4zi", "hi\x00*\x00\x00\x01\x00")
    --> =hi	65536	9

    unpack("!4c1Xlc1", "A***B")
    --> =A	B	6

    local function unpackError(...)
        if pcall(string.unpack, ...) then
            print("NO ERROR")
        end
    end

    unpackError("b", "")
    unpackError("i", "123")
    unpackError("bX", "2")
    unpackError("z", "abc")
    unpackError("c5", "abcd")
    unpackError("By", "a")
    unpackError("s1", "\x3ab")
end

do
    local function ps(f)
        print(string.packsize(f))
    end

    ps("bb")
    --> =2

    ps("c10i")
    --> =14

    ps("lx")
    --> =9

    ps("!4Bf")
    --> =8

    ps("!8hXi8")
    --> =8

    local function psError(f)
        if pcall(string.packsize, f) then
            print("NO ERROR")
        end
    end

    psError("c")
    psError("!20")
    psError("z")
    psError("s4")
end

do
    local function dl(f)
        return load(string.dump(f))
    end
    
    local function f(x)
        print(string.format("%s squared is %s", x, x*x))
    end

    print(type(string.dump(f)))
    --> =string

    dl(f)(2)
    --> =2 squared is 4

    dl(f)(10)
    --> =10 squared is 100

    local function g(x)
        return function(y)
            print("Working...")
            return x + y + 2
        end
    end

    print(dl(g)(3)(5))
    --> =Working...
    --> =10
end
