g = {}
setmetatable(g, {__gc = function() print"gone" end})

g2 = {}
setmetatable(g2, {__gc = function() print"gone 2" end})

a = ""
meta = {__gc = function(t) print(t.gc) end}

runtime.callcontext({kill={millis=1000}}, function()
    local x = {gc = "local cx"}
    setmetatable(x, meta)
    local y = {gc = "local cy"}
    setmetatable(y, meta)
    local z = {gc = "local cz"}
    setmetatable(z, meta)
    print"leave"
end)
print"after"
--> =leave
--> =local cz
--> =local cy
--> =local cx
--> =after

do
    local x = {gc = "local x"}
    setmetatable(x, meta)
    local y = {gc = "local y"}
    setmetatable(y, meta)
    local z = {gc = "local z"}
    setmetatable(z, meta)
    -- local t = {x, y}
end

-- When the runtime is closed, __gc metamethods should be called in reverse
-- order.
--> =local z
--> =local y
--> =local x
--> =gone 2
--> =gone
