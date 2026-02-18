-- config: reservedglobal
-- Tests for reserved global mode: "global" is a keyword, not a valid name.

-- Cannot use "global" as a local variable name
print(load("local global = 1"))
--> ~nil.*expected name near 'global'

-- Cannot use "global" as a local function name
print(load("local function global() end"))
--> ~nil.*expected name near 'global'

-- Cannot use "global" as a (plain) function name
print(load("function global() end"))
--> ~nil.*expected name near 'global'

-- Cannot assign to "global" as a variable
print(load("global = 1"))
--> ~nil.*expected name near '='

-- Global declarations still work normally
print(load("global x = 1") ~= nil)
--> =true

print(load("global function f() end") ~= nil)
--> =true

print(load("global *") ~= nil)
--> =true

print(load("global<const> x = 1") ~= nil)
--> =true
