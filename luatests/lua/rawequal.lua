print(rawequal(1, 1))
--> =true

print(rawequal(1, 2))
--> =false

local t = {}
print(rawequal(t, t))
--> =true

print(rawequal("abc", "abc"))
--> =true

print(rawequal(1.0, 1))
--> =true

print(rawequal(nil, nil))
--> =true

print(pcall(rawequal))
--> ~^false\t

print(pcall(rawequal, 1))
--> ~^false\t
