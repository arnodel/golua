d = string.dump(function()
    print(2, 3.5, 99999999, "hello", false, nil)
end)
load(d)()
--> =2	3.5	99999999	hello	false	nil

print(load("x == y"))
--> ~nil\t.*

-- Hexadecimal literals are truncated
print(0x12345678900000000000000000000ff)
--> =255

-- Decimal integer literals that do not fit into an integer are turned to
-- floats (specified in Lua 5.4)
print(5000000000000000000000000000000000000)
--> =5e+36

print(1e999999, -1e9999, 1e-99999)
--> =+Inf	-Inf	0.0

print("1e99999" + 0)
--> =+Inf

print("-2" + 0, "+2" + 0, "100000000000000000000" / "10000000000000000000")
--> =-2	2	10.0

-- Integer literals beyond 32 bits keep their value on every platform; on
-- 32-bit platforms they used to be inlined as their low 16 bits.
print(4294967296, 4294967301, 4611686018427387904, -4294967296)
--> =4294967296	4294967301	4611686018427387904	-4294967296

print(4294967296 == 1 << 32, 4611686018427387904 == 1 << 62)
--> =true	true
