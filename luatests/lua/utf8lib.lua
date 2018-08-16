do
    print(utf8.char(65, 66, 67))
    --> =ABC

    print(utf8.char(0x0394, 0x1000))
    --> =Δက

    print(utf8.char(0x65e5, 0x672c, 0x8a92))
    --> =日本誒
end

do
    for p, c in utf8.codes("ABéC") do
        print(p, c)
    end
    --> =1	65
    --> =2	66
    --> =3	233
    --> =5	67
end

do
    print(utf8.codepoint("ABC"))
    --> =65

    print(utf8.codepoint("ABC", 2))
    --> =66

    print(utf8.codepoint("日本誒", 1, -1))
    --> =26085	26412	35474
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
end
