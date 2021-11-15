quota = require'quota'


print(quota.rcall(1000, 1000, function() return 1 end))
--> =true	1

x = 0
print(quota.rcall(1000, 1000, function() while true do x = x + 1 end end))
--> =false

print(x > 10)
--> =true
