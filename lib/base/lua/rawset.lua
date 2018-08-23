local t = {}
rawset(t, "x", 42)
print(t.x)
--> =42

print(pcall(rawset, t, nil, 12), pcall(rawset, t, "y"), pcall(rawset, "a", "x", 42), 0)
--> =false	false	false	0
