-- This test file makes use of a custom "testudata" function that returns a new
-- userdata instance which prints a message when garbage collected.  See
-- runtime/lua_test.go for the implementation of testudata.

-- Resource are released when we leave a restricted context
print(runtime.callcontext({kill={cpu=1000}}, function()
    local c1 = testudata("c1")
    local c2 = testudata("c2")
    print"leave"
end))
--> =leave
--> =**release c2**
--> =**release c1**
--> =done

-- Resource are released when we leave a restricted context, even if it is killed
print(runtime.callcontext({kill={cpu=1000}}, function()
    local c3 = testudata("c3")
    local c4 = testudata("c4")
    while true do end
    print"leave"
end))
--> =**release c4**
--> =**release c3**
--> =killed

function withgc(s)
    local x = testudata(s)
    debug.setmetatable(x, {
        __gc=function() print("finalize " .. s) end,
        __close=function() print("close "  .. s) end,
    })
    return x 
end

-- Finalizers are run before resources are released
print(runtime.callcontext({kill={cpu=1000}}, function()
    local f1 = withgc("f1")
    local f2 = withgc("f2")
    print"leave"
end))
--> =leave
--> =finalize f2
--> =finalize f1
--> =**release f2**
--> =**release f1**
--> =done

-- If the context is killed, resources are released but finalizers do not run.
print(runtime.callcontext({kill={cpu=1000}}, function()
    local f3 = withgc("f3")
    local f4 = withgc("f4")
    while true do end
    print"leave"
end))
--> =**release f4**
--> =**release f3**
--> =killed

-- The finalizers run in the context, so it could kill them.
print(runtime.callcontext({kill={cpu=1000}}, function()
    local f5 = withgc("f5")
    local f6 = testudata("f6")
    debug.setmetatable(f6, {__gc=function()
        print"start f6"
        while true do end
    end})
    local f7 = withgc("f7")
    print"leave"
end))
--> =leave
--> =finalize f7
--> =start f6
-- At this point the context is killed

--> =**release f7**
--> =**release f6**
--> =**release f5**
--> =killed

-- In the case of to-be-closed variables, they are closed before their finalizer
-- is run (and so before the resource is released).
print(runtime.callcontext({kill={cpu=1000}}, function()
    local x1 <close> = withgc("x1")
    local x2 <close> = withgc("x2")
    print"leave"
end))
--> =leave
--> =close x2
--> =close x1
--> =finalize x2
--> =finalize x1
--> =**release x2**
--> =**release x1**
--> =done
