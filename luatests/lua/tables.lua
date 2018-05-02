local t = {3, 2, 1, x = "abc", ["zz"] = 12}
print(t[1])
--> =3
print(#t)
--> =3
t[4]=2
print(#t)
--> =4
print(t["x"] .. t.zz)
--> =abc12

t[6]=1
t[5]=1
print(#t)
--> =6

t[6]=nil
print(#t)
--> =5

t[4]=nil
t[5]=nil
print(#t)
--> =3
