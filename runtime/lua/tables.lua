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

t[3.2] = 5
print(t[3.2])
--> =5

t[5e2] = "hi"
print(t[500])
--> =hi

print(#t)
--> =3

t.xxx = nil
print(t.xxx)
--> =nil

print(pcall(function() t[nil] = 2 end))
--> ~false\ttable index is nil

do
    local t = {"x", "y"}
    local a, x = next(t)
    local b, y = next(t, a)
    if a < b then
        print(a..b, x..y)
    else
        print(b..a, y..x)
    end
    --> =12	xy

    print(next(t, b))
    --> =nil	nil

    print(pcall(next, t, "abc"))
    --> ~^false

    t[b] = nil
    print(next(t, a))
    --> =nil	nil
end
