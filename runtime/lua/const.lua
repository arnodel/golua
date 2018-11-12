d = string.dump(function()
    print(2, 3.5, "hello", false, nil)
end)
load(d)()
--> =2	3.5	hello	false	nil

