print(pcall(os.date))
--> ~true\t.*

local t = os.time({year=2008, month=1, day=11})
print(os.date("%Y-%m-%d", t))
--> =2008-01-11

local tt = os.date("*t", t)

print(tt.year .. "-" .. tt.month .. "-" .. tt.day)
--> =2008-1-11
