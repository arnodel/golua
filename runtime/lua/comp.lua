print(1 == 2, 3.2 == 3.2, 3.2 == 2, {} == 1, {} == {})
--> =false	true	false	false	false

print(1 < 2, 3.5 < -1, 2 < 1e10, 1e5 < 1e6, "xyz" < "xyza")
--> =true	false	true	true	true

print(1 <= 2, 3.5 <= -1, 2 <= 1e10, 1e5 <= 1e6, "xyz" <= "xyza")
--> =true	false	true	true	true

print(pcall(function() return {} <= {} end))
--> ~^false\t

print(pcall(function() return {} < {} end))
--> ~^false\t

local t = {}
local meta = {__lt=function(x, y) return not not y end}
setmetatable(t, meta)
print(t < true)
--> =true

-- Since Lua 5.4 the __le metamethod is no longer inferred from __lt if __le is
-- not implemented
print(pcall(function() return t <= false end))
--> ~false\t.*attempt to compare a table value with a boolean value

meta.__le = meta.__lt
print(t <= false)
--> =false
