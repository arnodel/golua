local callcc = require("cont").callcc

-- A context is run by Go code, so a continuation captured outside it cannot
-- be called inside it: it is not a way out of the context's limits.
do
    local k
    local done = {}
    callcc(function(c) k = c end)
    if not done.out then
        done.out = true
        print(runtime.callcontext({kill={cpu=10000}}, function() k() end))
        --> ~^error\t.*attempt to call a continuation across a Go call boundary
    end
end

-- Nor is one captured inside it a way back in.
do
    local k
    runtime.callcontext({kill={cpu=10000}}, function()
        callcc(function(c) k = c end)
    end)
    print(pcall(k))
    --> ~^false\t.*attempt to call a continuation across a Go call boundary
end

-- Calling a continuation uses CPU: a loop made of one ends like any loop.
print(runtime.callcontext({kill={cpu=10000}}, function()
    local k
    callcc(function(c) k = c end)
    k()
end))
--> =killed

-- Capturing and calling use memory, for the continuation and for the copies
-- of the frames it holds.
print(runtime.callcontext({kill={memory=100000}}, function()
    local k
    callcc(function(c) k = c end)
    k()
end))
--> =killed

-- The function is available to code that has to be CPU and memory safe.
print(runtime.callcontext({flags="cpusafe memsafe"}, function()
    local n = callcc(function(k) return k(1) + 1 end)
    return n
end))
--> =done	1
