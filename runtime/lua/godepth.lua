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
