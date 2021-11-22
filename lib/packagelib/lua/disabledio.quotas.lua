-- If IO is disabled, it's not possilbe to load a lua module from a file.

print(runtime.callcontext({io="off"}, pcall, require, "testlib.foo"))
--> =done	false	io disabled
