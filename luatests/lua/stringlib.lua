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
