print(select(1, 3, 2, 1))
--> =3

print(select(2, 3, 2, 1))
--> =2

print(select(4, 3, 2, 1))
--> =

do
    local function f(i, ...)
        return select(i, ...)
    end
    print(f(2, "a", "b", "c"))
--> =b
end

print(pcall(select, 0, 1, 2))
--> ~^false	.*
