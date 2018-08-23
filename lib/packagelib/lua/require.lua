print(require "testlib.foo") -- points at testlib/foo.lua
--> =foo
--> =bar

-- Second time, the module is already loaded so no side effects.
print(require "testlib.foo")
--> =bar

print(package.loaded["testlib.foo"])
--> =bar

print(require "testlib.bar") -- points at testlib/bar/init.lua
--> =42
