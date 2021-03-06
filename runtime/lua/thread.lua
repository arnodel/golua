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
