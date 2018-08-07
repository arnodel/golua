do
    print(utf8.char(65, 66, 67))
    --> =ABC

    print(utf8.char(0x0394, 0x1000))
    --> =Δက

    print(utf8.char(0x65e5, 0x672c, 0x8a92))
    --> =日本誒
end
