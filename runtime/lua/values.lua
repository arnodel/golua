print(1 + "2")
--> =3

print(1 + "3.2")
--> =4.2

print(pcall(function() return 1 + "3.a" end))
--> ~^false

print(pcall(function() return 1 + "x" end))
--> ~^false

print(string.char("65"))
--> =A

print(string.char("7e1"))
--> =F

print(pcall(string.char, "x"))
--> ~false.*integers

print(pcall(string.char, "7.1"))
--> ~false.*integers

print(string.byte("123", -1))
--> =51
