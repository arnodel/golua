print(pcall(package.searchpath))
--> ~^false\t.*2 arguments needed

print(pcall(package.searchpath, {}, "zzz"))
--> ~^false\t.*must be a string

print(pcall(package.searchpath, "zzz", {}))
--> ~^false\t.*must be a string

print(pcall(package.searchpath, "zzz", "aaa", {}))
--> ~^false\t.*must be a string

print(pcall(package.searchpath, "zzz", "aaa", "o", {}))
--> ~^false\t.*must be a string

print((package.searchpath("foo", "./?.lua")))
--> =nil

print((package.searchpath("testlib.foo", "./?.lua")))
--> =./testlib/foo.lua
