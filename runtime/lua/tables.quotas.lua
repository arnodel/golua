local quota = require 'quota'

-- Filling a table consumes memory
local t = {}
print(quota.rcall(0, 1000, function()
    local i = 1
    while true do
        t[i] = i
        i = i + 1
    end
end))
--> =false

print(#t > 10, #t < 100)
--> =true	true

-- Replacing scalar elements in a table doesn't consume memory
local t = {1}

quota.rcall(10000, 10000, function()
    local mem = quota.mem()
    local cpu = quota.cpu()
    print(quota.rcall(1000, 1000, function()
        while true do
            t[1] = t[1] + 1
        end
    end))
    --> =false

    -- Check we didn't run out of memory
    print(quota.mem() - mem < 500)
    --> =true

    -- Check we ran out of cpu
    print(quota.cpu() - cpu > 1000)
    --> =true

    -- Check we did a number of iterations
    print(t[1] > 100)
    --> =true
end)
