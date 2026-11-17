-- callcc(f) calls f(k), where k is the continuation of the call of callcc.
-- The cont library itself is tested in lib/contlib/lua/cont.lua.

local callcc = require("cont").callcc

-- Not calling k: callcc returns what f returns.
print(callcc(function(k) return 1, 2 end))
--> =1	2

-- k is a function, and a value like any other.
do
    local k = callcc(function(k) return k end)
    local t = {[k] = "key"}
    print(type(k), t[k], tostring(k))
    --> ~^function\tkey\tcontinuation: 0x[0-9a-f]+$
end

-- Calling k from inside f, from any depth, leaves f: an early exit.
do
    local function find(t, x)
        return callcc(function(k)
            local function walk(t)
                for _, v in ipairs(t) do
                    if type(v) == "table" then
                        walk(v)
                    elseif v == x then
                        k("found", v)
                    end
                end
            end
            walk(t)
            return "not found"
        end)
    end
    print(find({1, {2, {3, 4}}, 5}, 4))
    --> =found	4
    print(find({1, {2, {3, 4}}, 5}, 6))
    --> =not found
end

-- Calling k after callcc has returned makes it return again.  The state is
-- kept in a table: see below for what happens to a local variable.
do
    local k
    local count = {n = 0}
    local v = callcc(function(c) k = c; return 1 end)
    count.n = count.n + 1
    print(v, count.n)
    if v < 3 then
        k(v + 1)
    end
    --> =1	1
    --> =2	2
    --> =3	3
end

-- The same when the functions that led to the capture have returned since,
-- and their frames have been given to other calls.
do
    local k
    local function capture()
        return callcc(function(c) k = c; return 0 end)
    end
    local function next()
        local v = capture()
        return v + 1
    end
    local function noise(n)
        if n > 0 then
            return 1 + noise(n - 1)
        end
        return 0
    end
    local r = next()
    print(r, noise(10))
    if r < 3 then
        k(r)
    end
    --> =1	10
    --> =2	10
    --> =3	10
end

-- A frame is copied with what it has computed so far, which includes the
-- arguments of a call it is in the middle of making.
do
    local k
    local calls = {n = 0}
    local function show(...)
        print(...)
    end
    show("to Lua", calls.n, callcc(function(c) k = c; return 1 end))
    calls.n = calls.n + 1
    if calls.n < 3 then
        k(calls.n + 1)
    end
    --> =to Lua	0	1
    --> =to Lua	0	2
    --> =to Lua	0	3

    calls.n = 0
    print("to Go", calls.n, callcc(function(c) k = c; return 1 end))
    calls.n = calls.n + 1
    if calls.n < 3 then
        k(calls.n + 1)
    end
    --> =to Go	0	1
    --> =to Go	0	2
    --> =to Go	0	3

    -- The same for a tail call, to a Lua function and to a Go function.
    local function tail(f)
        return f("tail call", callcc(function(c) k = c; return 1 end))
    end
    for _, f in ipairs{show, print} do
        calls.n = 0
        tail(f)
        calls.n = calls.n + 1
        if calls.n < 3 then
            k(calls.n + 1)
        end
    end
    --> =tail call	1
    --> =tail call	2
    --> =tail call	3
    --> =tail call	1
    --> =tail call	2
    --> =tail call	3
end

-- A local variable that lives in a register is part of the frame: each time
-- the continuation is called it is as it was at the capture.  A variable
-- that a function captures and assigns lives in a box, which is shared.
do
    local k
    local tries = {n = 0}
    local reg, boxed = 0, 0
    local function bump() boxed = boxed + 1 end
    callcc(function(c) k = c end)
    reg = reg + 1
    bump()
    tries.n = tries.n + 1
    print(reg, boxed)
    if tries.n < 3 then
        k()
    end
    --> =1	1
    --> =1	2
    --> =1	3
end

-- Backtracking: amb returns one of its choices, and fail() goes back to the
-- most recent amb with choices left and makes it return the next one.  Each
-- continuation is called as many times as there are choices.
do
    local retries = {}
    local function fail()
        local retry = table.remove(retries)
        if retry == nil then
            error("no more choices")
        end
        return retry()
    end
    local function amb(choices)
        local i = 0
        return callcc(function(k)
            local function try()
                i = i + 1
                if i > #choices then
                    return fail()
                end
                retries[#retries + 1] = try
                return k(choices[i])
            end
            return try()
        end)
    end

    local found = {n = 0}
    local digits = {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13}
    local a = amb(digits)
    local b = amb(digits)
    local c = amb(digits)
    if a >= b or a * a + b * b ~= c * c then
        fail()
    end
    print(a, b, c)
    -- Asking for another solution is failing again.
    found.n = found.n + 1
    if found.n < 3 then
        fail()
    end
    --> =3	4	5
    --> =5	12	13
    --> =6	8	10
end

-- The frames a captured frame returns through are captured with it.  Here
-- there are fifty of them, each in the middle of an addition, and the
-- continuation is called three times.
do
    local k
    local calls = {n = 0}
    local function sum(n)
        if n == 0 then
            return callcc(function(c) k = c; return 0 end)
        end
        local rest = sum(n - 1)
        return n + rest
    end
    print(sum(50))
    calls.n = calls.n + 1
    if calls.n < 4 then
        k(calls.n * 1000)
    end
    --> =1275
    --> =2275
    --> =3275
    --> =4275
end

-- All the ways a function has of returning, to a frame that was captured:
-- its values, a tail call of a Lua function and of a Go function, values
-- followed by all the results of a call, and callcc itself.
do
    local k
    local calls = {n = 0}
    local function id(...)
        return ...
    end
    local function capture()
        return callcc(function(c) k = c; return 1 end)
    end
    local function values() local v = capture(); return v, v end
    local function luatail() local v = capture(); return id(v, "lua") end
    local function gotail() local v = capture(); return select(1, v, "go") end
    local function pushed() local v = capture(); return "pushed", id(v) end
    local function nested() local v = capture(); return callcc(function(c) return v, "callcc" end) end
    for _, f in ipairs{values, luatail, gotail, pushed, nested} do
        calls.n = 0
        print(f())
        calls.n = calls.n + 1
        if calls.n < 3 then
            k(calls.n + 1)
        end
    end
    --> =1	1
    --> =2	2
    --> =3	3
    --> =1	lua
    --> =2	lua
    --> =3	lua
    --> =1	go
    --> =2	go
    --> =3	go
    --> =pushed	1
    --> =pushed	2
    --> =pushed	3
    --> =1	callcc
    --> =2	callcc
    --> =3	callcc
end

-- A generator: each call of the function it returns runs walk up to its
-- next call of yield.  Two continuations take turns, the consumer's and the
-- walk's, and each is called once.
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

    local function leaves(tree)
        return generator(function(yield)
            local function walk(t)
                for _, v in ipairs(t) do
                    if type(v) == "table" then
                        walk(v)
                    else
                        yield(v)
                    end
                end
            end
            walk(tree)
        end)
    end

    local next = leaves{1, {2, {3, 4}}, {{5}}, 6}
    local out = {}
    while true do
        local v = next()
        if v == nil then
            break
        end
        out[#out + 1] = v
    end
    print(table.concat(out, " "))
    --> =1 2 3 4 5 6
end

-- An error in a copy of a frame says where it is, as in any frame.
do
    local k
    print(pcall(function()
        local n = callcc(function(c) k = c; return 1 end)
        if n < 2 then
            k(n + 1)
        end
        error("n is " .. n)
    end))
    --> ~false\t.*:\d+: n is 2

    -- So do the frames it returns to, which are not copied until it does:
    -- an error can be theirs, and a traceback goes through them.
    local trace
    local function leaf()
        local n = callcc(function(c) k = c; return 1 end)
        if n < 2 then
            k(n + 1)
        end
        trace = debug.traceback()
        error("the caller's fault", 2)
    end
    local function caller()
        leaf() -- the error is here
    end
    print(pcall(function() caller() end))
    --> ~^false\t.*:\d+: the caller's fault$
    print(select(2, trace:gsub("in function", "")) >= 3)
    --> =true
end

-- What is not copied is the Go code that waits for frames to finish: a
-- metamethod call, a library function that calls a function it is given.
-- So a continuation can only be called while the Go code it was captured
-- under is still waiting, and not from under more Go code.  For now this
-- includes leaving such Go code by calling a continuation captured outside
-- it.
do
    local k
    local done = {}

    -- Indexing this table with a function calls the function, from Go.
    local called = setmetatable({}, {__index = function(_, f) return f() end})

    local _ = called[function() callcc(function(c) k = c end) end]
    print(pcall(k))
    --> ~false\t.*attempt to call a continuation across a Go call boundary

    callcc(function(c) k = c end)
    if not done.meta then
        done.meta = true
        print(pcall(function() return called[k] end))
        --> ~false\t.*attempt to call a continuation across a Go call boundary
    end

    if not done.thread then
        done.thread = true
        print(coroutine.resume(coroutine.create(k)))
        --> ~false\t.*attempt to call a continuation from another thread
    end

    print(pcall(callcc))
    --> ~false\t.*bad argument #1
    print(pcall(callcc, 1))
    --> ~false\t.*attempt to call a number value
end

-- A protected call is not Go code that waits: the function that pcall calls
-- returns to a continuation like any other, which a captured continuation
-- holds.  So a continuation is a way out of a pcall, and a way back into one
-- that has returned.
do
    local k
    local count = {n = 0}

    -- Calls k with the given values, twice in all.
    local function again(...)
        count.n = count.n + 1
        if count.n < 3 then
            k(...)
        end
        count.n = 0
    end

    -- Out: it is callcc that returns, not pcall.
    print(callcc(function(c)
        pcall(function() c("out") end)
        return "not out"
    end))
    --> =out

    -- Back in: pcall returns again, with the values of this time.
    print(pcall(function()
        return "in", (callcc(function(c) k = c; return "first" end))
    end))
    --> =true	in	first
    --> =true	in	again
    --> =true	in	again
    again("again")

    -- The same when it is pcall that calls callcc: the continuation is then
    -- the protected call itself.
    print(pcall(callcc, function(c) k = c; return "first" end))
    --> =true	first
    --> =true	again
    --> =true	again
    again("again")

    -- And through two protected calls.
    print(pcall(pcall, callcc, function(c) k = c; return "first" end))
    --> =true	true	first
    --> =true	true	again
    --> =true	true	again
    again("again")

    -- An error after going back in stops at the pcall, each time.
    print(pcall(function()
        local v = callcc(function(c) k = c; return "first" end)
        error("failed " .. v, 0)
    end))
    --> =false	failed first
    --> =false	failed again
    --> =false	failed again
    again("again")

    -- It goes through the message handler of an xpcall.
    print(xpcall(function()
        local v = callcc(function(c) k = c; return "first" end)
        error("failed " .. v, 0)
    end, function(e) return "handled: " .. e end))
    --> =false	handled: failed first
    --> =false	handled: failed again
    --> =false	handled: failed again
    again("again")

    -- The protected calls that count towards the limit on their depth are
    -- those the program is in.  Leaving them by a continuation takes them
    -- off the count,
    for i = 1, 1500 do
        callcc(function(c) pcall(function() c() end) end)
    end
    print(pcall(function() return "not too deep" end))
    --> =true	not too deep

    -- also when it is pcall that calls the continuation,
    for i = 1, 1500 do
        callcc(function(c) pcall(c) end)
    end
    print(pcall(function() return "not too deep" end))
    --> =true	not too deep

    -- and going back into them puts them on it again: 990 deep, 20 more are
    -- too many, also when coming from outside all of them.
    local function nest(n, f)
        if n == 0 then
            return f()
        end
        return select(2, pcall(nest, n - 1, f))
    end
    print(nest(990, function()
        local v = callcc(function(c) k = c; return "first" end)
        return v .. ": " .. nest(20, function() return "not too deep" end)
    end))
    --> =first: stack overflow
    --> =again: stack overflow
    --> =again: stack overflow
    again("again")
end

-- A coroutine has a loop of its own, which stops at a yield and goes on at
-- the next resume: a continuation captured in a coroutine can be called in
-- it later on.
do
    local k

    print(coroutine.resume(coroutine.create(function()
        local n = callcc(function(c) k = c; return 1 end)
        if n < 3 then
            k(n + 1)
        end
        return n
    end)))
    --> =true	3

    -- After a yield and a resume.
    local co = coroutine.wrap(function()
        local n = callcc(function(c) k = c; return 1 end)
        coroutine.yield(n)
        if n < 3 then
            k(n + 1)
        end
        return "end"
    end)
    print(co(), co(), co(), co())
    --> =1	2	3	end

    -- After the coroutine was given a goroutine, here for a metamethod call
    -- (see runtime/lightthread.go).
    co = coroutine.wrap(function()
        local n = callcc(function(c) k = c; return 1 end)
        local _ = setmetatable({}, {__index = function() return 1 end}).x
        if n < 3 then
            k(n + 1)
        end
        return n
    end)
    print(co())
    --> =3
end
