-- If IO is disabled, it's not possilbe to load a lua module from a file.

print(runtime.callcontext({flags="iosafe"}, pcall, require, "testlib.foo"))
--> ~done\tfalse\t.*: could not find package 'testlib.foo'

-- Unless the module is in a directory granted by the fs quota

print(runtime.callcontext({fs={{dir="testlib", allowed="r"}}}, require, "testlib.foo"))
--> =foo
--> =done	bar
