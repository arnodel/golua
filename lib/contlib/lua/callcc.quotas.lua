local callcc = require("cont").callcc

-- A context is run by Go code, as a pcall is.  Calling a continuation
-- captured outside it leaves the context, which is then over: what runs next
-- runs in the context the continuation was captured in.
runtime.callcontext({kill={cpu=100000}}, function()
    local k
    local done = {}
    local used = runtime.context().used.cpu
    print(callcc(function(c) k = c; return "first" end))
    if not done.out then
        done.out = true
        runtime.callcontext({kill={cpu=10000}}, function()
            for i = 1, 1000 do end
            k("left the context")
        end)
        print("not reached")
    end
    --> =first
    --> =left the context

    -- What the context used is counted where it was started from.
    print(runtime.context().used.cpu - used >= 1000)
    --> =true
end)

-- A continuation captured inside a context is not a way back into it.
do
    local k
    runtime.callcontext({kill={cpu=10000}}, function()
        callcc(function(c) k = c end)
    end)
    print(pcall(k))
    --> ~^false\t.*attempt to call a continuation captured in a Go call that has returned
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
