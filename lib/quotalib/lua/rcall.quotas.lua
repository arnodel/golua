local quota = require'quota'

-- quota.rcall(cpuQuota, memQuota, f, [arg1, ...])
--
-- The quota.rcall function returns true and the f(arg1, ...) if that function
-- can be executed within the memory and cpu quota specified
print(quota.rcall(1000, 1000, function() return 1 end))
--> =true	1

local x = 0
-- Here the passed in function is an infinite loop, so execution will stop when
-- the budget of 1000 cpu is consumed.
print(quota.rcall(1000, 0, function() while true do x = x + 1 end end))
--> =false

-- Check that runtime managed to do a few iterations before being terminated
print(x > 10)
--> =true

function getquotas()
    local ucpu, qcpu = quota.cpu()
    local umem, qmem = quota.mem()
    return qcpu, qmem
end

print(getquotas())
--> =0	0

-- Test that nested rcalls work sensibly
quota.rcall(10000, 20000, function()
    print(getquotas())
    --> =10000	20000

    -- It's not possible to increase the quotas
    quota.rcall(20000, 30000, function()
        qcpu, qmem = getquotas()
        print(qcpu < 10000, qmem < 20000)
        --> =true	true
    end)

    -- CPU consumed inside the rcall is accounted for once the rcall has
    -- finished.
    cpuUsed = quota.cpu()

    -- It's possible to further decrease the quotas
    print(quota.rcall(5000, 5000, function()
        print(getquotas())
        --> =5000	5000
        while true do end
    end))
    --> =false

    print(quota.cpu() - cpuUsed >= 5000)
    --> =true

    -- Quotas get reset after
    print(getquotas())
    --> =10000	20000

    -- Memory consumed inside the rcall is accounted for once the rcall has
    -- finished.
    local memUsed = quota.mem()

    quota.rcall(10000, 20000, function()
        -- Consume some memory to check that will be accounted for outside rcall
        local s = ("a"):rep(10000)
    end)
    print(quota.mem() - memUsed >= 10000)
    --> =true
end)

-- Quotas get reset to their initial value
print(getquotas())
--> =0	0
