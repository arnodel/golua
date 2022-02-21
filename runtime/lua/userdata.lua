-- This test file makes use of a custom "testudata" function that returns a new
-- userdata instance which prints a message when garbage collected.  See
-- runtime/lua_test.go for the implementation of testudata.

u1 = testudata("u1") -- this will print the line "**release u1**" when collected
u2 = testudata("u2")

function withgc(s)
    local x = testudata(s)
    debug.setmetatable(x, {
        __gc=function() print("finalize " .. s) end,
    })
    return x 
end

-- Resources are released when the runtime is closed - their finalizers run
-- first
u3 = withgc("u3")
u4 = withgc("u4")

--> =finalize u4
--> =finalize u3
--> =**release u4**
--> =**release u3**

-- The following were defined at the top of the test file
--> =**release u2**
--> =**release u1**
