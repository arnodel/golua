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
--> =false	bar
--> =in function error (file [Go])
--> =in function xpcall (file [Go])
--> =in function <main chunk> (file luatest:27)
