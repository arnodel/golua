quota = require'quota'
string = require 'string'

s = "helLO"
s1000 = s:rep(1000)

-- string.lower consumes memory
print(quota.rcall(0, 4000, string.lower, s))
--> =true	hello

print(quota.rcall(0, 4000, string.lower, s1000))
--> =false


-- string.upper consumes memory

print(quota.rcall(0, 4000, string.upper, s))
--> =true	HELLO

print(quota.rcall(0, 4000, string.upper, s1000))
--> =false


-- string.reverse consumes memory

print(quota.rcall(0, 4000, string.reverse, s))
--> =true	OLleh

print(quota.rcall(0, 4000, string.reverse, s1000))
--> =false


-- string.sub consumes memory

print(quota.rcall(0, 1000, string.sub, s, 3, 2000))
--> =true	lLO

print(quota.rcall(0, 1000, string.sub, s1000, 3, 2000))
--> =false

-- string.byte consumes memory

-- helper function to consume the returned bytes from string.byte.
function len(s)
    return select('#', s:byte(1, #s))
end

print(len("foobar"))
--> =6

print(quota.rcall(0, 10000, len, s1000))
--> =false

-- string.char consumes memory

print(quota.rcall(0, 1000, string.char, s:byte(1, #s)))
--> =true	helLO

print(quota.rcall(0, 1000, string.char, s1000:byte(1, 1200)))
--> =false


-- string.rep consumes memory

print(quota.rcall(0, 1000, string.rep, "ha", 10))
--> =true	hahahahahahahahahaha

print(quota.rcall(0, 1000, string.rep, "ha", 600))
--> =false
