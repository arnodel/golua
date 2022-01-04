-- Filling a table consumes memory
local t = {}
print(runtime.callcontext({kill={memory=1000}}, function()
    local i = 1
    while true do
        t[i] = i
        i = i + 1
    end
end))
--> =killed

print(#t > 10, #t < 100)
--> =true	true

-- Replacing scalar elements in a table doesn't consume memory
local t = {1}

local ctx = runtime.callcontext(
    {kill={memory=10000, cpu=10000}},
    function()
        while true do
            t[1] = t[1] + 1
        end
    end
)

print(ctx.status)
--> =killed

-- Check we didn't run out of memory
print(ctx.used.memory < 1000)
--> =true

-- Check we ran out of cpu
print(ctx.used.cpu >= ctx.kill.cpu - 50)
--> =true

-- Check we did a number of iterations
print(t[1] > 100)
--> =true
