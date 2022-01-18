do
    local f = coroutine.wrap(function()
        coroutine.yield(1)
        coroutine.yield(2)
        return 3
    end)
    print(f(), f(), f())
    --> =1	2	3

    print(pcall(f))
    --> ~^false\t.*dead thread
end

do
    local co = coroutine.create(function ()
        local t = coroutine.running()
        print(coroutine.resume(t))
        --> ~^false\t.*running thread
        print(coroutine.status(t))
        --> =running
    end)
    coroutine.resume(co)
    print(coroutine.status(co))
    --> =dead
end

do
    print(pcall(coroutine.yield, 1))
    --> ~cannot yield from main thread
end

-- Closing a coroutine
do
    local co = coroutine.create(function() end)

    print(coroutine.close(co))
    --> =true

    -- A closed thread is dead
    print(coroutine.status(co))
    --> =dead

    -- It can be done again
    print(coroutine.close(co))
    --> =true

    -- Can't close the runnnig coroutine
    local main = coroutine.running()
    print(pcall(coroutine.close, main))
    --> ~false\t.*cannot close running thread

    local co = coroutine.create(function()
        print(pcall(coroutine.close, main))
        --> ~false\t.*cannot close normal thread
    end)
    coroutine.resume(co)

    -- Closing an errored thread returns the error
    local co = coroutine.create(error)

    print(coroutine.resume(co, 42))
    --> =false	42

    print(coroutine.close(co))
    --> =false	42
end

--
-- Coroutines and to-be-closed variables
--

function make(msg, err)
    t = {}
    setmetatable(t, {__close = function (x, e) 
        if e ~= nil then
            print(msg, e)
        else
            print(msg)
        end
        if err ~= nil then 
            error(err)
        end
    end})
    return t
end

do
    local co = coroutine.create(function ()
        local foo <close> = make("foo")
        coroutine.yield()
    end)
    coroutine.resume(co)
    print(coroutine.status(co))
    --> =suspended
    print(coroutine.close(co))
    -- Output from closing the "foo" var
    --> =foo
    -- Outcome of coroutine.close(co)
    --> =true
    print(coroutine.status(co))
    --> =dead
end

do
    local function f(n)
        local x <close> = make("x"..n)
        if n > 1 then
            f(n - 1)
        else
            coroutine.yield()
        end
    end
    local co = coroutine.create(f)
    coroutine.resume(co, 3)
    print(coroutine.status(co))
    --> =suspended
    print(coroutine.close(co))
    -- Output from closing the "x" vars
    --> =x1
    --> =x2
    --> =x3
    -- Outcome of coroutine.close(co)
    --> =true
    print(coroutine.status(co))
    --> =dead
end

-- Wrapping this in pcall to suppress the default message handler which is
-- adding traceback to error messages...
pcall(function()
    local function f(n)
        local x <close> = make("x"..n, "ERR"..n)
        if n > 1 then
            f(n - 1)
        else
            coroutine.yield()
        end
    end
    local co = coroutine.create(f)
    coroutine.resume(co, 3)
    print(coroutine.status(co))
    --> =suspended
    print(coroutine.close(co))
    -- Output from closing the "x" vars
    --> =x1
    --> ~x2\t.*ERR1
    --> ~x3\t.*ERR2
    -- Outcome of coroutine.close(co)
    --> ~false\t.*ERR3
    print(coroutine.status(co))
    --> =dead
end)
