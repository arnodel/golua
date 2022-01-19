-- Lua 5.4 introduces a 4th return value in the for in statement, which works as
-- a to be closed variable.

local foo = {}
setmetatable(foo, {__close=function() print"close foo" end})

local r3 = function(s, i)
    if i < 3 then return i + 1 else return nil end
end

for i in r3, nil, 0, foo do
    print(i)
end
print"done"
--> =1
--> =2
--> =3
--> =close foo
--> =done
