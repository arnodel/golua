print(rawget({x=1}, "x"))
--> =1

print(rawget({"hello"}, 1.0))
--> =hello

print(rawget({}, "abc"))
--> =nil

print(pcall(rawget, "hello", 1))
--> ~^false\t.*#1 must be a table

print(pcall(rawget, {}))
--> ~^false\t
