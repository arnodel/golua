-- A function that produces an error
function foo(x) bar(x) end
function bar(x) baz(x) end
function baz(x) error(x) end

-- A dummy error handler
function bye() return "bye" end

print(xpcall(foo, print, "hello"))
--> =hello
--> =false	nil

print(xpcall(foo, bye, "hello"))
--> =false	bye

-- pcall within xpcall

print(xpcall(pcall, print, foo, "hello"))
--> =true	false	hello

-- xpcall within pcall

print(pcall(xpcall, foo, print, "hello"))
--> =hello
--> =true	false	nil

-- Nested xpcalls
print(xpcall(
    function()
        print(xpcall(foo, bye, "hi"))
        foo("bonjour")
    end,
    print
))
--> =false	bye
--> =bonjour
--> =false	nil

-- error in message handler, we eventually bail out
print(xpcall(error, error, "foo"))
--> =false	error in error handling

-- It tries handling the error at least 10 times but at most 100 times

n = 0
function err(e)
    n = n + 1
    error(e)
end

print(xpcall(error, err, "hello"))
--> =false	error in error handling
print(n >= 10 and n <= 100)
--> =true

-- Error handlers are just called once.
t = {}
debug.setmetatable(t, {__index=function() error("abc") end})

function p(x)
    print("debug", x)
    return x
end

print(xpcall(function() return t[1] end, p))
--> =debug	abc
--> =false	abc
