-- Creating a named vararg table consumes memory even without arguments: the
-- table header and the storage for n are charged.
local function f(...args) return args end
print(runtime.callcontext({kill={memory=10000}}, function()
    for i = 1, 1000 do f() end
end))
--> =killed

-- Expanding a vararg table whose n was raised builds n values. They are
-- charged before they are allocated, both in a table constructor and in a
-- call.
local function g(...args) args.n = 1 << 22 local t = {...} return #t end
print(runtime.callcontext({kill={memory=100000}}, g))
--> =killed

local function count(...) return select("#", ...) end
local function h(...args) args.n = 1 << 22 return count(...) end
print(runtime.callcontext({kill={memory=100000}}, h))
--> =killed

-- The expansion is charged CPU too: n for the values it builds, on top of what
-- using them costs afterwards (about n again).  A limit cannot show this, as
-- the later cost kills the call too, so compare what the context used.
local function u(...args) args.n = 1 << 20 return count(...) end
local ctx = runtime.callcontext({kill={cpu=100000000}}, u)
print(ctx.status, ctx.used.cpu > 3 * (1 << 20) // 2)
--> =done	true
