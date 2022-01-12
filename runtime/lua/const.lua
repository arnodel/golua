d = string.dump(function()
    print(2, 3.5, 99999999, "hello", false, nil)
end)
load(d)()
--> =2	3.5	99999999	hello	false	nil

print(load("x == y"))
--> ~nil\t.*

print(1e999999, -1e9999, 1e-99999)
--> =+Inf	-Inf	0

print("1e99999" + 0)
--> =+Inf

print("-2" + 0, "+2" + 0)
--> =-2	2
