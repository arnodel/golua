-- A call with listed arguments makes the callee's frame at the call, so the
-- callee is read then, after the arguments (astcomp compileCall).

-- The arguments are evaluated before a value that cannot be called is
-- found out.
local h
local ok, err = pcall(function() return h(print("arg"), 1) end)
print(ok, err:match("attempt to call a nil value"))
--> =arg
--> =false	attempt to call a nil value

-- A captured callee reassigned while an argument is evaluated: the call
-- uses the function the variable held before.
local function first(x) return "first" .. x end
local function other(x) return "other" .. x end
local f = first
local function swap() f = other; return "!" end
print(f(swap()))
--> =first!
print(f(1))
--> =other1

-- The same for a method's receiver.
local o1 = {name = "o1"}
function o1.m(self, x) return self.name .. x end
local o2 = {name = "o2", m = o1.m}
local o = o1
local function swapo() o = o2; return "!" end
print(o:m(swapo()))
--> =o1!

-- A fixed captured callee is read at the call without a copy.
local g = function(a, b) return a + b end
local function callg() return g(1, 2) end
print(callg())
--> =3

-- A callee with no arguments is read at the call, whether fixed or not.
local k = function() return "k1" end
local function callk() return k() end
print(callk())
--> =k1
k = function() return "k2" end
print(callk())
--> =k2
