-- A best effort attempt is made to name anonymous functions
local function myname() print(debug.getinfo(2).name) end

x = function() myname() end
x()
--> =x

t = {}
t.foo = function() myname() end
t.foo()
--> =foo

t = {
    hello = function() myname() end
}
t.hello()
--> =hello
