local function hook() end

do
    debug.sethook(hook, "clr", 10)
    print(debug.gethook())
    --> ~function.*\tcrl\t10

    debug.sethook()
    print(debug.gethook())
    --> =nil		0
end

do
    local co = coroutine.create(function() end)
    debug.sethook(co, hook, "cr", -8)
    print(debug.gethook(co))
    --> ~function.*\tcr\t0

    debug.sethook(co)
    print(debug.gethook(co))
    --> =nil		0
end

-- Errors
do
    local co = coroutine.create(function() end)

    print(pcall(debug.gethook, false))
    --> ~false\t.*#1 must be a thread

    print(pcall(debug.sethook, 123))
    --> ~false\t.*#1 must be a thread

    print(pcall(debug.sethook, co, hook))
    --> ~false\t.*3 arguments needed

    print(pcall(debug.sethook, co, hook, {}))
    --> ~false\t.*#3 must be a string

    print(pcall(debug.sethook, hook, "cr", "hello"))
    --> ~false\t.*#3 must be an integer

end