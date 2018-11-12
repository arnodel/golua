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
setmetatable(t, {__lt=function(x, y) return not not y end})
print(t < true, t <= false)
--> =true	false
