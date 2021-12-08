print(-(1+1))
--> =-2

print(-1.01)
--> =-1.01

print(pcall(function(x) return -x end, {}))
--> ~^false\t

print(1 + 1.2, 2.2 + 1, 3.2 + 1.2)
--> =2.2	3.2	4.4

local function testbin(f, x, y)
    print(not pcall(f, x, y) and not pcall(f, y, x))
end

testbin(function(x, y) return x + y end, 1, {})
--> =true

print(1 - 2, 3.5 - 2, 4 - 1.5, 4.5 - 2.5)
--> =-1	1.5	2.5	2
 
testbin(function(x, y) return x - y end, 1, {})
--> =true

print(1 * 1, 1.5 * 3, 2 * 2.5, 3.0 * 6.0)
--> =1	4.5	5	18

testbin(function(x, y) return x * y end, 1, {})
--> =true

print(4 / 2, 2 / 0.5, 4.0 / 2, 1.5 / 0.5)
--> =2	4	2	3

testbin(function(x, y) return x / y end, 1, {})
--> =true

print(4 // 2, 4.5 // 2, 2 // 0.8, 3.5 // 0.5)
--> =2	2	2	7

testbin(function(x, y) return x // y end, 1, {})
--> =true

print(5 % 2, 3.5 % 2, 2 % 0.5, 3.5 % 0.5)
--> =1	1.5	0	0

print(-3 % 2, -3.0 % 2)
--> =1	1

testbin(function(x, y) return x % y end, 1, {})
--> =true

print(3^2, 9^0.5, 0.5^2, 4.0^1.5)
--> =9	3	0.25	8

testbin(function(x, y) return x^y end, 1, {})
--> =true

print(pcall(function() return 1//0 end))
--> ~false\t.*divide by zero

print(pcall(function() return 1%0 end))
--> ~false\t.*perform 'n%0'