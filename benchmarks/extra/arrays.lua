-- arrays.lua
--
-- Building and walking arrays the ways Lua programs do: appending with
-- t[#t + 1] and with table.insert, a stack pushed and popped with
-- table.remove, filling and reading by index, ipairs, and the length
-- operator throughout.  Written for golua's benchmarks; it returns a function
-- that does all of this n times over arrays of 200 numbers and returns a
-- checksum.

return function(n)
    local insert, remove = table.insert, table.remove
    local sum = 0
    for round = 1, n do
        -- Appended by length.
        local a = {}
        for i = 1, 200 do
            a[#a + 1] = (i * i + round) % 1000
        end
        -- The even ones, appended with table.insert.
        local b = {}
        for _, v in ipairs(a) do
            if v % 2 == 0 then
                insert(b, v)
            end
        end
        -- A stack: pushed, then popped until it is empty.
        local s = {}
        for i = 1, #b do
            s[#s + 1] = b[i]
        end
        local top = 0
        while #s > 0 do
            top = top + remove(s)
        end
        -- Filled by index, read by index.
        local c = {}
        for i = 1, #a do
            c[i] = a[i] + (b[i] or 0)
        end
        for i = 1, #c do
            sum = sum + c[i]
        end
        sum = (sum + top + #b) % 1000000007
    end
    return tostring(sum)
end
