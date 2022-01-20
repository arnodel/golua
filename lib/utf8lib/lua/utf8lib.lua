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

    -- Encoding is lax since Lua 5.4

    print(pcall(utf8.char, 0x7fffffff, "hello", 100))
    --> ~false	.*should be an integer

    print(pcall(utf8.char, 0x80000000, "hello", 100))
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

    -- lax mode (Lua 5.4)
    for p, c in utf8.codes("\u{200000}\u{3FFFFFF}\u{4000000}\u{7FFFFFFF}", true) do
        print(p, string.format("%x", c))
    end
    --> =1	200000
    --> =6	3ffffff
    --> =11	4000000
    --> =17	7fffffff

    print(pcall(utf8.codes, "abc", "123"))
    --> ~false\t.*must be a boolean
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

    -- Check "non-strict" unicode (Lua 5.4)
    print(pcall(utf8.codepoint, "\u{7FFFFFFF}"))
    --> ~false\t.*invalid UTF-8 code

    print(utf8.codepoint("\u{7FFFFFFF}", 1, 1, true) == 0x7FFFFFFF)
    --> =true
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

    print(utf8.len("abcd", 5))
    --> =0

    local err = errtest(utf8.len)

    err()
    --> ~value needed

    err(123)
    --> ~must be a string

    err("ABC", {})
    --> ~must be an integer

    err("ABC", 2, {})
    --> ~must be an integer

    err("abc", 0, 2)
    --> ~out of range

    err("abc", 1, 4)
    --> ~out of range

    -- lax mode (Lua 5.4)

    err("sdfjl", 2, 4, "true")
    --> ~must be a boolean

    print(utf8.len("\u{200000}\u{3FFFFFF}\u{4000000}\u{7FFFFFFF}", 1, -1, true))
    --> =4

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
