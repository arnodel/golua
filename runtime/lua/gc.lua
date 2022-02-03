g = {}
setmetatable(g, {__gc = function() print"gone" end})

g2 = {}
setmetatable(g2, {__gc = function() print"gone 2" end})

a = ""
meta = {__gc = function(t) print(t.gc) end}

do
    local x = {gc = "x"}
    setmetatable(x, meta)
    local y = {gc = "y"}
    setmetatable(y, meta)
    local z = {gc = "z"}
    setmetatable(z, meta)
    -- local t = {x, y}
end


-- When the runtime is closed, __gc metamethods should be called in reverse
-- order.
--> =z
--> =y
--> =x
--> =gone 2
--> =gone
