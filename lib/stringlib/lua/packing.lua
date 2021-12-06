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

    pack("<J", 1000)
    --> =232	3	0	0	0	0	0	0

    pack(">I10", 9876543210)
    --> =0	0	0	0	0	2	76	176	22	234

    pack("<I10", 9876543210)
    --> =234	22	176	76	2	0	0	0	0	0

    pack("<I6", 123456789)
    --> =21	205	91	7	0	0

    pack(">I6", 123456789)
    --> =0	0	7	91	205	21

    pack("i8", 123)
    --> =123	0	0	0	0	0	0	0

    pack("i9", -1)
    --> =255	255	255	255	255	255	255	255	255

    pack("I4", 1122334455)
    --> =247	118	229	66

    local function packError(...)
        if pcall(string.pack, ...) then
            print("NO ERROR")
        end
    end

    packError() -- value needed
    packError(123) -- string expected
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
    packError("z", "a\0b") -- string contains zero
    packError("f", "xx") -- expect float
    packError("i") -- expect a value
    packError("s") -- expect a value
    packError("s", {}) -- expect a string-like
    packError("f", 1e50) -- float too big
    packError("i1", 300) -- int too big
    packError("c18446744073709551620", "hello") -- size to big
    packError("c18446744073709551619", "hello") -- size to big

    s = string.rep("x", 500)
    print(#s)
    --> =500
    packError("s1", s) -- string too long

    print("DONE")
    --> =DONE
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

    unpack("<hh", "\x01\x00\x00\x01")
    --> =1	256	5

    unpack("<J", "\x00\x00\x01\x00\x00\x00\x00\x00")
    --> =65536	9

    unpack(">I", "\xff\xff\xff\xff")
    --> =4294967295	5

    unpack("<f", "\x00\x00\x00\x00")
    --> =0	5

    unpack("<d", "\x00\x00\x00\x00\x00\x00\x00\x00")
    --> =0	9

    unpack("<I8", "\x00\x00\x01\x00\x00\x00\x00\x00")
    --> =65536	9

    unpack("<I9", "\x00\x00\x01\x00\x00\x00\x00\x00\x00")
    --> =65536	10

    unpack("<i8", "\x00\x00\x01\x00\x00\x00\x00\x00")
    --> =65536	9

    unpack("<i9", "\x00\x00\x01\x00\x00\x00\x00\x00\x00")
    --> =65536	10

    unpack("<i10", "\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff")
    --> =-1	11

    unpack(">i10", "\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff")
    --> =-1	11

    unpack("<i6", "\xff\xff\x00\x00\x00\x00")
    --> =65535	7

    unpack(">i6", "\x00\x00\x00\x00\xff\xff")
    --> =65535	7

    unpack("<i3>i3", "\xff\xff\xff\xff\xff\xff")
    --> =-1	-1	7

    unpack("z", "ho\0")
    --> =ho	4

    local function unpackError(...)
        if pcall(string.unpack, ...) then
            print("NO ERROR")
        end
    end

    unpackError() -- 2 arguments needed
    unpackError("a") -- 2 argument needed
    unpackError(1, "z") -- #1 must be a string
    unpackError("b", {}) -- #2 must be a string
    unpackError("a", "xyz", 40) -- #3 out of range
    unpackError("b", "")
    unpackError("i", "123")
    unpackError("bX", "2")
    unpackError("z", "abc")
    unpackError("c5", "abcd")
    unpackError("By", "a")
    unpackError("s1", "\x3ab")
    unpackError("XX", "hello")
    unpackError("X ", "hello")
    unpackError("Xz", "a\0") -- z has no alignment
    unpackError("!4zf", "a\0") -- no padding
    unpackError("I6", "1234") -- not enough bytes to read
    unpackError("<I10", "12345678xx") -- ext bytes should be 0
    unpackError(">I10", "\0") -- missing 0 byte
    unpackError(">i10", "") -- not enough sign bytes
    unpackError(">i10", "1100000000") -- wrong value for sign byte
    unpackError(">i10", "\0\0\xff\xff\xff\xff\xff\xff\xff\xff") -- too big
    unpackError(">i10", "\xff\xff\0\0\0\0\0\0\0\0") -- too big negative
    unpackError("<i6", "123") -- not enough bytes
    unpackError(">i6", "123") -- not enough bytes
end

do
    local function ps(f)
        print(string.packsize(f))
    end

    ps("bb")
    --> =2

    ps("c10i")
    -- Was 14 but moved packsize of i to 8
    -- --> =14
    --> =18

    ps("lx")
    --> =9

    ps("!4Bf")
    --> =8

    ps("!8hXi8")
    --> =8

    local function psError(...)
        if pcall(string.packsize, ...) then
            print("NO ERROR")
        end
    end

    psError() -- value needed
    psError(false) -- value must be a string
    psError("c")
    psError("!20")
    psError("z")
    psError("s4")
end

do
    local function pu(f, ...)
        print(string.unpack(f, string.pack(f, ...)))
    end

    pu("<i>i=i", 1, 2, 3)
    --> ~1	2	3	

end