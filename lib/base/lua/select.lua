print(select(1, 3, 2, 1))
--> ~^3\t2\t1$

print(select(2, 3, 2, 1))
--> ~^2\t1$

print(select(4, 3, 2, 1))
--> =

do
    local function f(i, ...)
        return select(i, ...)
    end
    print(f(2, "a", "b", "c"))
--> ~^b\tc$
end

print(pcall(select, 0, 1, 2))
--> ~^false	.*

print(select('#', 1, 2, 3))
--> =3

print(pcall(select))
--> ~false\t.*value needed

print(pcall(select, 'hello', 1, 2, 3))
--> ~false\t.*integer or '#'

print(select(-1, 1, 2 , 3, 4))
--> =4
