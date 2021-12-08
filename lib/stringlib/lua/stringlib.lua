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

    local errbyte = errtest(string.byte)

    errbyte()
    --> ~value needed

    errbyte({})
    --> ~must be a string

    errbyte("xxx", true)
    --> ~must be an integer

    errbyte("xxx", 1, nil)
    --> ~must be an integer
end

do
    print(string.char(65, 66, 67))
    --> =ABC

    local errchar = errtest(string.char)

    errchar(-1)
    --> ~out of range

    errchar(256)
    --> ~out of range

    errchar(1, 2, "x")
    --> ~must be integers
end

do
    print(string.len("abc"), string.len(""))
    --> =3	0

    errtest(string.len)()
    --> ~value needed

    errtest(string.len)(123)
    --> ~must be a string
end

do
    local s = "ABCdef123"
    print(s:lower())
    --> =abcdef123

    print(s:upper())
    --> =ABCDEF123

    errtest(string.lower)()
    --> ~value needed

    errtest(string.lower)(123)
    --> ~must be a string

    errtest(string.upper)()
    --> ~value needed

    errtest(string.upper)(123)
    --> ~must be a string
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

    local errrep = errtest(string.rep)

    errrep()
    --> ~2 arguments needed

    errrep(1, 2)
    --> ~must be a string

    errrep("xx", {})
    --> ~must be an integer

    errrep("xx", -1)
    --> ~out of range

    errrep("x", 2, 1)
    --> ~must be a string

    errrep("xxxxxxxxxx", 1 << 62)
    --> ~overflow

    errrep("xx", 1 << 62, ";")
    --> ~overflow
end

do
    local s = "EGASSEM TERCES"
    print(s:reverse())
    --> =SECRET MESSAGE

    print(string.reverse("12345"))
    --> =54321

    errtest(string.reverse)()
    --> ~value needed

    errtest(string.reverse)(123)
    --> ~must be a string
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

    print(s:sub(2, 3))
    --> =bc

    print(s:sub(3, 6))
    --> =c

    local subtest = errtest(string.sub)

    subtest()
    --> ~2 arguments needed

    subtest(1, 2)
    --> ~must be a string

    subtest("x", {})
    --> ~must be an integer

    subtest("xxx", 1, true)
    --> ~must be an integer
end

do
    local function pf(...)
        print(string.format(...))
    end
    
    local errf = errtest(string.format)

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

    pf("%c", 65)
    --> =A

    errf("%c")
    --> ~not enough values

    errf("%c", {})
    --> ~invalid value

    errf("%d", "hello")
    --> ~invalid value

    pf("-%u-", 55)
    --> =-55-

    pf("%i", -12)
    --> =-12

    pf("%x", 255)
    --> =ff

    errf("%e")
    --> ~not enough values

    errf("%e", false)
    --> ~invalid value

    errf('"%s"')
    --> ~not enough values

    local t = {}
    setmetatable(t, {__tostring=function() error("bad", 0) end})
    errf("%s", t)
    --> =bad

    errf("%q")
    --> ~not enough values

    pf("[%q]", nil)
    --> =[nil]

    errf("%q", {})
    --> ~no literal

    pf("%q %q %q", false, 1, 1.5)
    --> =false 1 1.5

    errf("%t")
    --> ~not enough values

    pf("This is %t", true)
    --> =This is true

    errf("%t", 1)
    --> ~invalid value

    errf("%z")
    --> ~invalid format string

    -- Not enough values is not OK
    print(pcall(pf, "%s %d", 1))
    --> ~^false\t.*$

    errf()
    --> ~value needed

    errf(321)
    --> ~must be a string

end

do
    local function dl(...)
        return load(string.dump(...))
    end
    
    local function f(x)
        print(string.format("%s squared is %s", x, x*x))
    end

    print(type(string.dump(f)))
    --> =string

    dl(f)(2)
    --> =2 squared is 4

    dl(f, true)(10)
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

    local errd = errtest(string.dump)

    errd()
    --> ~value needed

    errd(true)
    --> ~must be a lua function
end
