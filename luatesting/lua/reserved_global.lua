-- config: reservedglobal
-- Tests that the -- config: directive correctly configures the runtime.

print(not load("global = 1"))
--> =true

print(load("global x = 1") ~= nil)
--> =true
