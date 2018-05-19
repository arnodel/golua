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
