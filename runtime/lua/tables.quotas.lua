-- Filling a table consumes memory (each growth of its storage is charged
-- what it allocates: 10000 bytes hold a little over 100 integers)
local t = {}
print(runtime.callcontext({kill={memory=10000}}, function()
    local i = 1
    while true do
        t[i] = i
        i = i + 1
    end
end))
--> =killed

print(#t > 50, #t < 500)
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

-- A table that outlives the context it grows in is charged to the contexts
-- around it, also when the growth is what terminates the context
do
    local function fill(t)
        local n = #t
        for i = n + 1, n + 1000000 do
            t[i] = i
        end
    end

    -- The nested context has what is left of the memory of the outer one, so
    -- the outer one does not outlive it
    local t = {}
    local rounds = 0
    print(runtime.callcontext({kill={memory=100000}}, function()
        for i = 1, 5 do
            rounds = rounds + 1
            runtime.callcontext({}, fill, t)
        end
    end))
    --> =killed
    print(rounds, #t < 10000)
    --> =1	true

    -- The same two contexts down
    t = {}
    rounds = 0
    print(runtime.callcontext({kill={memory=100000}}, function()
        for i = 1, 5 do
            rounds = rounds + 1
            runtime.callcontext({}, function()
                runtime.callcontext({}, fill, t)
            end)
        end
    end))
    --> =killed
    print(rounds, #t < 10000)
    --> =1	true

    -- A nested context with a limit of its own is charged the growth that
    -- terminates it, and so is the outer one
    t = {}
    rounds = 0
    print(runtime.callcontext({kill={memory=100000}}, function()
        for i = 1, 5 do
            rounds = rounds + 1
            runtime.callcontext({kill={memory=30000}}, fill, t)
        end
    end))
    --> =killed
    print(rounds < 5, #t < 10000)
    --> =true	true
end
