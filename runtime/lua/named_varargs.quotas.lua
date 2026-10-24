-- Creating a named vararg table consumes memory even without arguments: the
-- table header and the storage for n are charged.
local function f(...args) return args end
print(runtime.callcontext({kill={memory=10000}}, function()
    for i = 1, 1000 do f() end
end))
--> =killed
