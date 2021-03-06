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
    print(utf8.char(65, 66, 67))
    --> =ABC

    print(utf8.char(0x0394, 0x1000))
    --> =Δက

    print(utf8.char(0x65e5, 0x672c, 0x8a92))
    --> =日本誒

    print(pcall(utf8.char, 65, "hello", 100))
    --> ~false	.*should be an integer


    print(pcall(utf8.char, -1, "hello", 100))
    --> ~false	.*out of range

    print(pcall(utf8.char, 0x110000, "hello", 100))
    --> ~false	.*out of range
    
end

do
    for p, c in utf8.codes("ABéC") do
        print(p, c)
    end
    --> =1	65
    --> =2	66
    --> =3	233
    --> =5	67

    print(pcall(utf8.codes))
    --> ~false	.*value needed

    print(pcall(utf8.codes, 123))
    --> ~false	.*must be a string

    local iter = utf8.codes("A\xff")
    print(iter())
    --> =1	65

    print(pcall(iter))
    --> ~false	.*invalid UTF-8 code
end

do
    print(utf8.codepoint("ABC"))
    --> =65

    print(utf8.codepoint("ABC", 2))
    --> =66

    print(utf8.codepoint("日本誒", 1, -1))
    --> =26085	26412	35474

    local err = errtest(utf8.codepoint)

    err()
    --> ~value needed

    err({})
    --> ~must be a string

    err("ABC", false)
    --> ~must be an integer

    err("ABC", 1, {})
    --> ~must be an integer

    err("XYZ", -5, 3)
    --> ~out of range

    err("XYZ", 1, 10)
    --> ~out of range

    err("ab\xff", 3)
    --> ~invalid UTF-8 code
end

do
    print(utf8.len("ABC"))
    --> =3

    print(utf8.len("abcdef", 3))
    --> =4

    print(utf8.len("123456", 2, 4))
    --> =3

    print(utf8.len("日本誒"))
    --> =3

    print(utf8.len("日本誒", 2))
    --> =nil	2

    local err = errtest(utf8.len)

    err()
    --> ~value needed

    err(123)
    --> ~must be a string

    err("ABC", {})
    --> ~must be an integer

    err("ABC", 2, {})
    --> ~must be an integer
end

do
    local s = "abé日本誒"
    local function test(...)
        local ok, offset = pcall(utf8.offset, s, ...)
        if ok then
            print(offset)
        else
            print("ERROR")
        end
    end

    test(1)
    --> =1

    test(2, 3)
    --> =5

    test(5)
    --> =8

    test(1, 4)
    --> =ERROR

    test(10)
    --> =nil
    
    test(0, 4)
    --> =3

    test(0, 7)
    --> =5

    test(-2)
    --> =8

    test(-3, 8)
    --> =2

    test(-10)
    --> =nil

    test(-1, 7)
    --> =ERROR

    local err = errtest(utf8.offset)

    err()
    --> ~2 arguments needed

    err("ABC")
    --> ~2 arguments needed

    err({}, 1)
    --> ~must be a string

    err("ABC", "X")
    --> ~must be an integer

    err("BAC", 1, "Y")
    --> ~must be an integer

    err("ABC", 2, -4)
    --> ~out of range

    err("ABC", 2, 5)
    --> ~out of range

end
