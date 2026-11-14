-- A context that is killed has nothing left to close its pending variables
-- with: they are let go of, and the ones pending outside it are not.
do
    local outside <close> = setmetatable({}, {
        __close = function() print("closed outside the context") end,
    })
    print(runtime.callcontext({kill={cpu=1000}}, function()
        local inside <close> = setmetatable({}, {
            __close = function() print("not closed") end,
        })
        for _ in pairs({}) do end
        while true do end
    end))
    --> =killed
end
--> =closed outside the context
