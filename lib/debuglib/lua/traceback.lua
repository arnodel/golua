print(debug.traceback())
--> =in function <main chunk> (file luatest:1)


print(debug.traceback("foo"))
--> =foo
--> =in function <main chunk> (file luatest:5)

function foo()
    print(debug.traceback("in a function"))
end
foo()
--> =in a function
--> =in function foo (file luatest:10)
--> =in function <main chunk> (file luatest:12)

function bar()
    pcall(foo)
end
bar()
--> =in a function
--> =in function foo (file luatest:10)
--> =in function pcall (file [Go])
--> =in function bar (file luatest:18)
--> =in function <main chunk> (file luatest:20)

print(xpcall(error, debug.traceback, "bar"))
--> ~false\t.*: bar
--> =in function error (file [Go])
--> =in function xpcall (file [Go])
--> =in function <main chunk> (file luatest:27)

function foo(x) bar(x) end
function bar(x) baz(x) end
function baz(x) print(debug.traceback(nil, x)) end

foo(1)
--> =in function baz (file luatest:35)
--> =in function bar (file luatest:34)
--> =in function foo (file luatest:33)
--> =in function <main chunk> (file luatest:37)

foo(2)
--> =in function bar (file luatest:34)
--> =in function foo (file luatest:33)
--> =in function <main chunk> (file luatest:43)

foo(3)
--> =in function foo (file luatest:33)
--> =in function <main chunk> (file luatest:48)

-- We run out of stack here:
foo(10)
--> =

function cofoo()
    cobar()
end

function cobar()
    coroutine.yield(1)
end

co = coroutine.create(cofoo)
print(coroutine.resume(co))
--> =true	1

print(debug.traceback(co))
--> =in function cobar (file luatest:61)
--> =in function cofoo (file luatest:57)

print(debug.traceback({}))
--> ~table:.*

print(pcall(debug.traceback, "foo", "bar"))
--> ~false\t.*#2 must be an integer