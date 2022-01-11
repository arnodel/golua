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
