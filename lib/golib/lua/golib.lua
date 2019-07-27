print("hello")
--> =hello

print(hello)
--> =world

print(double(2))
--> =4

print(polly.Age)
--> =10

print(polly.Name)
--> =Polly

print(polly.Descr())
--> =age: 10, name: Polly

print(ben.Descr())
--> =age: 5, name: Ben

print(ben.Age)
--> =5

benben = ben.Mix(ben)
print(benben.Descr())
--> =age: 10, name: Ben-Ben

ben.Age = 7
print(ben.Age)
--> =7

benalice = ben.Mix{Name = "Alice", Age = 3}
print(benalice.Descr())
--> =age: 10, name: Ben-Alice

print(mapping.answer)
--> =42

print(mapping["foo"])
--> =nil

mapping["foo"] = 128
print(mapping.foo)
--> =128

print(slice[2])
--> =here

slice[1] = "was"

print(slice[0], slice[1], slice[2])
--> ~I\twas\there

print(sprintf("the %s is %d", "answer", 6*7))
--> =the answer is 42

print(twice(function(n) return 2*n end)(5))
--> =20

-- do
--     go = require("golib")
--     fmt = go.import("fmt")
--     sprintf = fmt.Sprintf
--     print(sprintf("-%s-", "hello"))
-- end
-- --> =-hello-
