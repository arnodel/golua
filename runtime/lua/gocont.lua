local function apply(f, ...)
    return f(...)
end

apply(print, 1, 2)
--> =1	2

print(apply(rawlen, "abc"))
--> =3

print(pcall(apply, rawequal, 1))
--> ~^false

local function errtest(...)
    local ok, err = pcall(...)
    if ok then
        print "OK"
    else
        print(err)
    end
end

print(type(string.dump(apply)))
--> =string

print(math.sin(0))
--> =0

print(string.byte("A", 1))
--> =65

errtest(string.dump, 1)
--> ~must be a lua function

errtest(string.len, 1)
--> ~must be a string

errtest(string.len)
--> ~value needed

errtest(coroutine.create, "wrong!")
--> ~must be a callable

errtest(coroutine.resume, {})
--> ~must be a thread

errtest(string.byte, "x", {})
--> ~must be an integer

errtest(math.cos, true)
--> ~must be a number

errtest(rawget, 1, 2)
--> ~must be a table

errtest(dofile, 1, 2, 3)
--> ~must be a string
