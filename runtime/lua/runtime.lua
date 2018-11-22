local debug = require"debug"

local function metastring(instance, val)
    debug.setmetatable(instance, {__tostring=function() return val end})
    print(instance)
    debug.setmetatable(instance, nil)
end

metastring(nil, "NIL")
--> =NIL

metastring("hello", "bonjour")
--> =bonjour

metastring(1, "one")
--> =one

metastring(1e3, "mille")
--> =mille

metastring(true, "false")
--> =false

metastring({}, "{}")
--> ={}

print(getmetatable({}))
--> =nil

local t = {}
setmetatable(t, {x=24})
print(getmetatable(t).x)
--> =24

setmetatable(t, {__metatable={x=42}})
print(getmetatable(t).x)
--> =42

