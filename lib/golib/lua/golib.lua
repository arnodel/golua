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
