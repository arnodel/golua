-- Tests for measures agains irrecoverable stack overflows

-- Recursive table.sort
do
    local n = 0
    local x = {}
    setmetatable(x, {__lt=function(x, y)
        n = n + 1
        table.sort{x, y}
    end})
    print(pcall(table.sort, {x, x}))
    --> ~false\t.*stack overflow
    print(n <= 1000 and n >= 900)
    --> =true
end

-- Recursive string.gsub
do
    local n = 0
    local function f()
        n = n + 1
        string.gsub("x", ".", f)
    end
    print(pcall(f))
    --> ~false\t.*stack overflow
    print(n <= 1000 and n >= 900)
    --> =true
end

-- Metamethods called by the VM itself run in nested interpreter loops with
-- no GoFunction in between.  A metamethod that triggers its own event must
-- hit a stack overflow error, not grow the Go stack without bound.
do
    local n
    local function deep(f)
        n = 0
        local ok, err = pcall(f)
        return ok, tostring(err):find("stack overflow") ~= nil, n >= 900 and n <= 1000
    end

    local mt = {}
    mt.__index = function(t, k) n = n + 1 return t[k] end
    print(deep(function() return setmetatable({}, mt).x end))
    --> =false	true	true

    mt = {}
    mt.__newindex = function(t, k, v) n = n + 1 t[k] = v end
    print(deep(function() setmetatable({}, mt).x = 1 end))
    --> =false	true	true

    mt = {}
    mt.__eq = function(a, b) n = n + 1 return a == b end
    print(deep(function() return setmetatable({}, mt) == setmetatable({}, mt) end))
    --> =false	true	true

    mt = {}
    mt.__lt = function(a, b) n = n + 1 return a < b end
    print(deep(function() local a = setmetatable({}, mt) return a < a end))
    --> =false	true	true

    mt = {}
    mt.__le = function(a, b) n = n + 1 return a <= b end
    print(deep(function() local a = setmetatable({}, mt) return a <= a end))
    --> =false	true	true

    mt = {}
    mt.__add = function(a, b) n = n + 1 return a + b end
    print(deep(function() return setmetatable({}, mt) + 1 end))
    --> =false	true	true

    mt = {}
    mt.__unm = function(a) n = n + 1 return -a end
    print(deep(function() return -setmetatable({}, mt) end))
    --> =false	true	true

    mt = {}
    mt.__len = function(a) n = n + 1 return #a end
    print(deep(function() return #setmetatable({}, mt) end))
    --> =false	true	true

    mt = {}
    mt.__concat = function(a, b) n = n + 1 return a .. b end
    print(deep(function() return setmetatable({}, mt) .. "x" end))
    --> =false	true	true

    mt = {}
    mt.__close = function()
        n = n + 1
        local x <close> = setmetatable({}, mt)
    end
    print(deep(function() local x <close> = setmetatable({}, mt) end))
    --> =false	true	true
end

