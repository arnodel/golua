-- To-be-closed variables and continuations.

local callcc = require("cont").callcc

-- closing(name) is a value that says so when it is closed.
local function closing(name)
    return setmetatable({}, {
        __close = function(_, err) print("close " .. name, err) end,
    })
end

-- Calling a continuation leaves blocks, as break and return do: their
-- variables are closed, the innermost first, and no error is given.
do
    print(callcc(function(k)
        local a <close> = closing("a")
        do
            local b <close> = closing("b")
            k("left")
        end
    end))
    --> =close b	nil
    --> =close a	nil
    --> =left
end

-- Only the blocks that are left: the one the continuation carries on in
-- keeps its variable until it ends.
do
    local a <close> = closing("a")
    print(callcc(function(k)
        local b <close> = closing("b")
        k("left b")
    end))
    print("still in a")
    --> =close b	nil
    --> =left b
    --> =still in a
end
--> =close a	nil

-- The same from functions called in the block, however deep.
do
    local function deep(n, k)
        local v <close> = closing("deep " .. n)
        if n == 0 then
            k("left")
        end
        return deep(n - 1, k) + 1
    end
    print(callcc(function(k) return deep(2, k) end))
    --> =close deep 0	nil
    --> =close deep 1	nil
    --> =close deep 2	nil
    --> =left
end

-- The closing value of a for loop is one of these.
do
    print(callcc(function(k)
        for i in function(_, i) return i + 1 end, nil, 0, closing("loop") do
            if i == 3 then
                k("left at " .. i)
            end
        end
    end))
    --> =close loop	nil
    --> =left at 3
end

-- A block in which a continuation was captured can be entered again, by
-- calling that continuation.  Leaving it does not close its variable, which
-- is closed when the block ends: here is a generator whose walk keeps a
-- variable open from one value to the next, as a coroutine would.
do
    local function generator(walk)
        local consumer, walker
        local function yield(v)
            callcc(function(k)
                walker = k
                consumer(v)
            end)
        end
        return function()
            return callcc(function(k)
                consumer = k
                if walker then
                    walker()
                end
                walk(yield)
                consumer(nil)
            end)
        end
    end

    local next = generator(function(yield)
        local source <close> = closing("source")
        yield(1)
        yield(2)
        print("walk ends")
    end)
    print(next())
    print(next())
    print(next())
    --> =1
    --> =2
    --> =walk ends
    --> =close source	nil
    --> =nil

    -- The walk and its consumer can each have a variable, in blocks of which
    -- neither is inside the other: each is closed when its own block ends.
    next = generator(function(yield)
        local w <close> = closing("the walk's")
        yield(1)
        yield(2)
    end)
    print(next())
    do
        local c <close> = closing("the consumer's")
        print(next())
        print(next())
    end
    --> =1
    --> =2
    --> =close the walk's	nil
    --> =nil
    --> =close the consumer's	nil
end

-- A block that is entered several times ends several times, and its variable
-- is closed once.
do
    local k
    local calls = {n = 0}
    do
        local once <close> = closing("once")
        callcc(function(c) k = c end)
        calls.n = calls.n + 1
        print("block ends", calls.n)
    end
    if calls.n < 3 then
        k()
    end
    --> =block ends	1
    --> =close once	nil
    --> =block ends	2
    --> =block ends	3
end

-- Leaving a block by an error closes its variable whether a continuation was
-- captured in it or not, as it always did.
do
    local k
    print(pcall(function()
        local v <close> = closing("failed")
        callcc(function(c) k = c end)
        error("oops", 0)
    end))
    --> =close failed	oops
    --> =false	oops
end

-- Under a pcall that a continuation leaves, everything is closed: there is
-- no entering it again.
do
    print(callcc(function(k)
        pcall(function()
            local v <close> = closing("in pcall")
            callcc(function(inner) end)
            k("left the pcall")
        end)
    end))
    --> =close in pcall	nil
    --> =left the pcall
end

-- An error in closing is the error of the call of the continuation.
do
    print(pcall(function()
        callcc(function(k)
            local v <close> = setmetatable({}, {__close = function() error("cannot close", 0) end})
            k("not reached")
        end)
    end))
    --> =false	cannot close

    -- Also when a Go function makes the call.
    print(pcall(function()
        callcc(function(k)
            local v <close> = setmetatable({}, {__close = function() error("cannot close", 0) end})
            testtailcall(k, "not reached")
        end)
    end))
    --> =false	cannot close

    -- A value which has lost its way of being closed since it was declared
    -- is such an error.
    print(pcall(function()
        callcc(function(k)
            local mt = {__close = function() end}
            local v <close> = setmetatable({}, mt)
            mt.__close = nil
            k("not reached")
        end)
    end))
    --> ~^false\t.*to be closed value missing a __close metamethod
end
