local callcc = require("cont").callcc

-- A context is not left by calling a continuation, as other Go code that
-- waits for Lua code is: the code that starts a context gets it back,
-- whatever runs in it.  Calling a continuation captured outside the context
-- ends it with an error.
do
    local k
    local done = {}
    print(callcc(function(c) k = c; return "first" end))
    if not done.out then
        done.out = true
        print(runtime.callcontext({kill={cpu=10000}}, function()
            k("not reached")
        end))
        -- A pcall in the context does not stop the call, and the context
        -- does.
        print(runtime.callcontext({kill={cpu=10000}}, function()
            pcall(k, "not reached")
            return "nor this"
        end))
        print("the contexts are over")
    end
    --> =first
    --> ~^error\t.*attempt to leave a context by calling a continuation
    --> ~^error\t.*attempt to leave a context by calling a continuation
    --> =the contexts are over
end

-- Inside a context continuations are what they are outside one.
print(runtime.callcontext({kill={cpu=10000}}, function()
    return callcc(function(k) pcall(k, "out of a pcall, in the context") end)
end))
--> =done	out of a pcall, in the context

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
