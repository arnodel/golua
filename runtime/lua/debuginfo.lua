-- Test the DebugInfo() method of continuations

function foo()
    print(debug.traceback("in a function"))
end
foo()
--> =in a function
--> =in function foo (file luatest:4)
--> =in function <main chunk> (file luatest:6)

local t = {}
debug.setmetatable(t, {__tostring=function()
    print(debug.traceback("TOSTRING"))
    return "HAHA"
end})

print(t)
--> =TOSTRING
--> =in function __tostring (file luatest:13)
--> =in function tostring (file [Go])
--> =in function print (file [Go])
--> =in function <main chunk> (file luatest:17)
--> =HAHA

-- Test debug info for a function

print(debug.getinfo(print).name)
--> =print
print(debug.getinfo(foo).name)
--> =foo
