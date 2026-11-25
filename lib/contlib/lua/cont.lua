-- The cont library.  What continuations do is tested where they are made,
-- in runtime/lua/continuation.lua.

-- The library is not a global: a program asks for it.
print(cont, callcc)
--> =nil	nil

local cont = require("cont")
print(type(cont), require("cont") == cont)
--> =table	true

for name, f in pairs(cont) do
    print(name, type(f))
end
--> =callcc	function

-- callcc(f) calls f(k) and returns what f returns, unless k is called, which
-- makes callcc return the arguments of k.
print(cont.callcc(function(k) return "returned" end))
--> =returned
print(cont.callcc(function(k) k("called", type(k)) end))
--> =called	function
