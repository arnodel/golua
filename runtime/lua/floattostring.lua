-- Floats convert to strings as in Lua 5.5 (tostringbuffFloat in lobject.c):
-- "%.15g", or "%.17g" if that does not read back as the same float, then
-- ".0" if the result looks like an integer.  The expected lines are the
-- output of PUC Lua 5.5.1.

print(999999.0, 1234567.0, -1234567.0, 4294967294.0)
--> =999999.0	1234567.0	-1234567.0	4294967294.0

print(1e14, 1e15, 1e16, 1e100)
--> =100000000000000.0	1e+15	1e+16	1e+100

print(0.1, 123.456, 1e-4, 1e-5, 3.14159265358979)
--> =0.1	123.456	0.0001	1e-05	3.14159265358979

-- "%.15g" does not read back: "%.17g"
print(1/3, 0.1 + 0.2, 100 * 1.1)
--> =0.33333333333333331	0.30000000000000004	110.00000000000001

print(2^53, 2^63, 18446744073709551614.0)
--> =9007199254740992.0	9.2233720368547758e+18	1.8446744073709552e+19

print(-0.0, 1/0, -1/0)
--> =-0.0	inf	-inf

-- The sign of a NaN depends on the platform, as with C's printf.
print(0/0)
--> ~^-?nan$

-- Every conversion of a float to a string agrees.
local x = 1234567.0
print(tostring(x), x .. "", string.format("%s", x), table.concat({x, 1/3}, " "))
--> =1234567.0	1234567.0	1234567.0	1234567.0 0.33333333333333331

-- Exercise the same values through each conversion, including io.write.
local cases = {
    {1234567.0, "1234567.0"},
    {1e14, "100000000000000.0"},
    {1e15, "1e+15"},
    {1e100, "1e+100"},
    {0.1, "0.1"},
    {-0.0, "-0.0"},
    {2^53, "9007199254740992.0"},
    {math.huge, "inf"},
    {-math.huge, "-inf"},
    {1234567, "1234567"}, -- Integer formatting stays unchanged.
}
local nan = 0/0
local nantext = tostring(nan)
assert(nantext == "nan" or nantext == "-nan")
cases[#cases + 1] = {nan, nantext}
local output = io.output()
local f = assert(io.tmpfile())
io.output(f)
for _, case in ipairs(cases) do
    local value, expected = case[1], case[2]
    assert(tostring(value) == expected)
    assert(value .. "" == expected)
    assert(string.format("%s", value) == expected)
    assert(table.concat({value}) == expected)
    assert(io.write(value, "\n"))
end
io.output(output)
assert(f:seek("set"))
for _, case in ipairs(cases) do
    assert(f:read("l") == case[2])
end
assert(f:close())

-- The text reads back as the same float.
for _, y in ipairs({1/3, 2^53, 0.1 + 0.2, 1e300, -5e-324, 2.2250738585072014e-308}) do
    if tonumber(tostring(y)) ~= y then print("not read back", y) end
end
print("ok")
--> =ok
