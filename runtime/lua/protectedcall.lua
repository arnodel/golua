-- pcall and xpcall hand over to the function they call: no Go code waits for
-- it (runtime/protectedcall.go).  These are the things that depend on that,
-- or that it could have broken.

local function msg(e) return type(e) == "string" and (e:match(":%d+: (.*)") or e) or e end

-- A coroutine can yield inside pcall and xpcall, and be resumed into them.
do
    local co = coroutine.wrap(function()
        print(pcall(function()
            local x = coroutine.yield("in pcall")
            error(x, 0)
        end))
        print(xpcall(function()
            return coroutine.yield("in xpcall")
        end, print))
        return "end"
    end)
    print(co())
    --> =in pcall
    print(co("resumed with an error"))
    --> =false	resumed with an error
    --> =in xpcall
    print(co("resumed"))
    --> =true	resumed
    --> =end

    -- The message handler can yield too.
    co = coroutine.wrap(function()
        return xpcall(error, function(e) return e .. coroutine.yield("in the handler") end, "a", 0)
    end)
    print(co())
    --> =in the handler
    print(co("b"))
    --> =false	ab
end

-- The message handler is that of the xpcall the error happens in, not of
-- the xpcall that started last: here a coroutine is suspended in one when
-- the error happens outside it.
do
    local co = coroutine.wrap(function()
        xpcall(coroutine.yield, function(e) return "the coroutine's: " .. e end)
    end)
    print(xpcall(function()
        co()
        error("x", 0)
    end, function(e) return "the caller's: " .. e end))
    --> =false	the caller's: x

    -- A pcall under way in a suspended coroutine does not hide the handler
    -- either.
    co = coroutine.wrap(function() pcall(coroutine.yield) end)
    print(xpcall(function()
        co()
        error("x", 0)
    end, function(e) return "handled: " .. e end))
    --> =false	handled: x
end

-- The handler is called where the error is, through the Go functions that
-- are waiting for the code that failed, and through a resume.
do
    local function where(e) return msg(e) .. " in " .. debug.getinfo(2, "n").name end
    local function thrower() error("x") end
    print(xpcall(function() table.sort({2, 1}, thrower) end, where))
    --> =false	x in error
    print(xpcall(coroutine.wrap(thrower), where))
    --> =false	x in error

    -- pcall inside takes over: the outer handler is not called.
    local called = false
    print(xpcall(function() return pcall(thrower) end, function() called = true end))
    --> ~true\tfalse\t.*: x
    print(called)
    --> =false
end

-- An error in closing a variable when the function returns is caught, with
-- the values the function had already returned dropped.
do
    print(pcall(function()
        local x <close> = setmetatable({}, {__close = function() error("in close", 0) end})
        return 1, 2, 3
    end))
    --> =false	in close
end

-- A handler that fails is given its own error, a limited number of times.
-- The count starts again with each protected call: many errors caught one
-- after the other are not "error in error handling".
do
    local tries = 0
    print(xpcall(error, function(e) tries = tries + 1 error(e, 0) end, "x"))
    --> =false	error in error handling
    print(tries)
    --> =10

    local caught = 0
    for i = 1, 50 do
        local ok, e = xpcall(error, function(e) return e end, i)
        if not ok and e == i then caught = caught + 1 end
    end
    print(caught)
    --> =50
end

-- Protected calls can nest to a limit: a function that calls itself through
-- pcall sees an error, and the calls return one by one.
do
    local depth = 0
    local function rec()
        depth = depth + 1
        local ok, e = pcall(rec)
        if not ok then error(e, 0) end
    end
    print(pcall(rec))
    --> ~false\t.*stack overflow
    print(depth)
    --> =1000

    -- The depth goes down again when calls return and when errors are
    -- caught, so that a long loop of them is no overflow.
    local n = 0
    for i = 1, 3000 do
        if pcall(error) == false and pcall(type, i) then n = n + 1 end
    end
    print(n)
    --> =3000
end

-- pcall in tail position does not keep the caller's frame.
do
    local function loop(i)
        if i == 0 then return "done" end
        return select(2, pcall(loop, i - 1))
    end
    print(loop(500))
    --> =done
end

-- Hooks see pcall return after the function it calls.
do
    local log = {}
    local function f() end
    debug.sethook(function(ev) log[#log + 1] = ev:sub(1, 1) .. ":" .. tostring(debug.getinfo(2, "n").name) end, "cr")
    pcall(f)
    pcall(error)
    debug.sethook()
    print(table.concat(log, " "))
    --> =r:sethook c:pcall c:f r:f r:pcall c:pcall c:error r:error r:pcall c:sethook
end

-- A message handler that itself goes through protected calls and fails
-- still has a limited number of tries: the count is of the handler calls an
-- error is in, whatever is caught in between.  (The handler here is the
-- function that failed, which overflows again each time.)
do
    local function loop()
        assert(pcall(loop))
    end
    local ok, e = xpcall(loop, loop)
    print(ok, e)
    --> =false	error in error handling
end
