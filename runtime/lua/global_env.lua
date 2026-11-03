-- "global x = value" checks that x isn't defined by indexing _ENV, as any
-- access to a global does: _ENV needn't be a table, and its __index
-- metamethod counts.

print(pcall(load("local _ENV = 1 global x = 2")))
--> ~false\t.*attempt to index a number value

print(pcall(load("local _ENV = nil global x = 2")))
--> ~false\t.*attempt to index a nil value

local log = {}
local env = setmetatable({}, {__index = function(t, k) log[#log + 1] = k return 1 end})
print(pcall(load("global x = 2", "chunk", "t", env)))
--> ~false\t.*global 'x' already defined

print(log[1])
--> =x

env = setmetatable({}, {__index = function() return nil end})
print(pcall(load("global y = 2 return y", "chunk", "t", env)))
--> =true	2
