local function ipairs_iterator(t, n)
    if n < #t then
        return n + 1, t[n + 1]
    end
end

local function _ipairs(t)
    return ipairs_iterator, t, 0
end

local t = {5, 4, 3}
local s = 0

for i, v in _ipairs(t) do
    s = s + i * v
end
print(s) -- 5*1 + 4*2 + 3*3
--> =22
