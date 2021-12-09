local function aFunction(...)
    local i = debug.getinfo(...)
    if not i then
        print("none")
    else
        print(i.name, i.currentline, i.source)
    end
end

local function foo(...)
    aFunction(...)
    return 1 -- avoid TCO
end

foo(1)
--> =aFunction	2	luatest

foo(2)
--> =foo	11	luatest

foo(0)
--> =getinfo	0	[Go]

foo(10)
--> =none

print(pcall(debug.getinfo))
--> ~false\t.*: bad argument #1.*

foo(foo)
--> =foo	-1	luatest

-- Get a thread
function cofoo()
    cobar()
end

function cobar()
    coroutine.yield(1)
end

co = coroutine.create(cofoo)
print(coroutine.resume(co))
--> =true	1

-- Check getinfo in the thread

print(pcall(foo, co))
--> ~false\t.*: missing argument: f.*

foo(co, 1)
--> =cobar	39	luatest

foo(co, 2)
--> =cofoo	35	luatest

print(pcall(foo, co, true))
--> ~false\t.*

foo(co, 1.0)
--> =cobar	39	luatest

print(pcall(foo, co, 1.5))
--> ~false\t.*
