quota = require'quota'

-- quota.rcall(cpuQuota, memQuota, f, [arg1, ...])
--
-- The quota.rcall function returns true and the f(arg1, ...) if that function
-- can be executed within the memory and cpu quota specified
print(quota.rcall(1000, 1000, function() return 1 end))
--> =true	1

x = 0
-- Here the passed in function is an infinite loop, so execution will stop when
-- the budget of 1000 cpu is consumed.
print(quota.rcall(1000, 0, function() while true do x = x + 1 end end))
--> =false

-- Check that runtime managed to do a few iterations before being terminated
print(x > 10)
--> =true
