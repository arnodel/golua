function _pairs(t)
    return next, t, nil
end

local t = {x=1, y=2, z="hello"}
local u = {}
for k, v in _pairs(t) do
    u[k] = v
end
print(u.x, u.y, u.z)
--> =1	2	hello

u = {}
for k, v in pairs(t) do
    u[k] = v .. "!"
end
print(u.x, u.y, u.z)
--> =1!	2!	hello!
