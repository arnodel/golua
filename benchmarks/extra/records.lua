-- records.lua
--
-- Tables as records and as dictionaries: records made by a constructor with
-- fields, a dictionary indexed by strings that are computed rather than
-- written in the source, records and dictionaries walked with pairs, and
-- fields updated through a computed key.  Written for golua's benchmarks; it
-- returns a function that does all of this n times over 100 records and
-- returns a checksum.

local names = {"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta"}

return function(n)
    local total = 0
    for round = 1, n do
        -- Records made by a constructor.
        local points = {}
        for i = 1, 100 do
            points[i] = {x = i, y = i * 2, z = round, tag = names[i % #names + 1]}
        end
        -- A dictionary that counts them by tag.
        local count = {}
        for i = 1, #points do
            local k = points[i].tag
            count[k] = (count[k] or 0) + 1
        end
        -- Every field of every record, then the dictionary, with pairs.
        for _, p in ipairs(points) do
            for k, v in pairs(p) do
                if k ~= "tag" then
                    total = total + v
                end
            end
        end
        for k, v in pairs(count) do
            total = total + v * #k
        end
        -- A record built field by field, updated through a computed key.
        local acc = {}
        for i = 1, #names do
            acc[names[i]] = 0
        end
        for i = 1, #points do
            local p = points[i]
            acc[p.tag] = acc[p.tag] + p.x
        end
        for i = 1, #names do
            total = total + acc[names[i]]
        end
        total = total % 1000000007
    end
    return tostring(total)
end
