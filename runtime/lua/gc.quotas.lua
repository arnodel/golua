local c = 0
local meta = {__gc = function() c = c + 1 end}
local function mk()
    local t = {}
    setmetatable(t, meta)
    return t
end

-- Resource constraints forces finalizers to be called inside the context
runtime.callcontext({kill = {cpu = 1000000}}, function()
    x = mk()
    -- No resource constraints do not
    runtime.callcontext({}, function()
        y = mk()
    end)
    print(c)
    --> =0
end)
print(c)
--> =2


local meta = {__gc = function(t) print(t.gc) end}

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
