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