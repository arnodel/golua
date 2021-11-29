-- setmetatable

do
    local t = {}
    local meta = {__tostring=function() return "meta" end}

    debug.setmetatable(t, meta)
    print(t)
    --> =meta

    debug.setmetatable(t, nil)
    print(t)
    --> ~table:.*

    print(pcall(debug.setmetatable, t, false))
    --> =false	#2 must be a table

    print(pcall(debug.setmetatable, t))
    --> =false	2 arguments needed
end
