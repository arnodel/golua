print(require "testlib.foo")
--> =foo
--> =bar

-- Second time, the module is already loaded so no side effects.
print(require "testlib.foo")
--> =bar
