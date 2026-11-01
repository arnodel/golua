-- A call evaluates the callee (and self for a method), then the arguments,
-- then calls.  A captured variable read for the call keeps the value it had
-- when it was read, even if a later argument reassigns it.

local x = 1
local function g() x = 2 return 0 end
local function f(a, b, c) return a end
print(f(x, g(), 3))
--> =1

local h = function() return "old" end
local function k() h = function() return "new" end return 0 end
print(h(k()))
--> =old

local o = {name = "o"}
local p = {name = "p"}
function o:who() return self.name end
p.who = function() return "p's method" end
local function swap() o = p return 0 end
print(o:who(swap()))
--> =o

-- The arguments are evaluated before calling a nil value fails.  (Not when
-- the last argument is a call or "...": its values are pushed one by one into
-- a continuation made first.)
local n
local evaluated = false
local function mark() evaluated = true end
print(pcall(function() n(mark(), 0) end))
--> ~false\t.*attempt to call a nil value
print(evaluated)
--> =true
