assert(1 + 1 == 2)
assert(0)
assert(true)
assert("")
-- No output

print(pcall(assert, 1 + 1 == 3))
--> =false	assertion failed!

print(pcall(assert, false, "xx"))
--> =false	xx

print(pcall(assert, nil, 123))
--> =false	123

print(pcall(assert))
--> ~false\t.*value needed