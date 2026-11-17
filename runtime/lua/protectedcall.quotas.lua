-- A context that is terminated inside protected calls: they do not catch
-- the termination, and are not counted as under way afterwards.
do
    local function spin() while true do end end
    local n = 0
    for i = 1, 1100 do
        local ctx = runtime.callcontext({kill = {cpu = 200}}, pcall, pcall, spin)
        if ctx.status == "killed" then n = n + 1 end
    end
    print(n)
    --> =1100
    -- 1100 abandoned pcalls, twice over, would be past the limit.
    print(pcall(pcall, pcall, print, "still fine"))
    --> =still fine
    --> =true	true	true
end

-- The handler of an xpcall outside a context is not called for an error
-- inside the context, which returns it.
do
    local called = false
    print(xpcall(function()
        local ctx, err = runtime.callcontext({}, error, "x", 0)
        return ctx.status, err
    end, function() called = true end))
    --> =true	error	x
    print(called)
    --> =false
end
