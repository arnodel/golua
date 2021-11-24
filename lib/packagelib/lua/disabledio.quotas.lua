-- If IO is disabled, it's not possilbe to load a lua module from a file.

print(runtime.callcontext({flags="iosafe"}, pcall, require, "testlib.foo"))
--> =done	false	missing flags: iosafe
