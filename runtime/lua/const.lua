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
--> =+Inf	-Inf	0

print("1e99999" + 0)
--> =+Inf

print("-2" + 0, "+2" + 0, "100000000000000000000" / "10000000000000000000")
--> =-2	2	10
