print(tonumber(123))
--> =123

print(tonumber("321"))
--> =321

print(tonumber("+12"))
--> =12

print(tonumber("-12.45"))
--> =-12.45

print(tonumber("xx"))
--> =nil

print(tonumber("ff", 16))
--> =255

print(tonumber("-Z", 36))
--> =-35

print(tonumber("   -1001", 2))
--> =-9

print(tonumber("012"))
--> =12

print(tonumber("0xffffffffffffffff"))
--> =-1

print(tonumber("-0xffffffffffffffff"))
--> =1

print(tonumber("+102", 3))
--> =11

print(tonumber("ff", 15))
--> =nil

print(tonumber("[]", 11))
--> =nil