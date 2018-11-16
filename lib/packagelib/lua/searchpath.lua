print(pcall(package.searchpath))
--> ~^false\t.*2 arguments needed

print(pcall(package.searchpath, 1, 2))
--> ~^false\t.*must be a string

print(pcall(package.searchpath, "zzz", 2))
--> ~^false\t.*must be a string

print(pcall(package.searchpath, "zzz", "aaa", 3))
--> ~^false\t.*must be a string

print(pcall(package.searchpath, "zzz", "aaa", "o", {}))
--> ~^false\t.*must be a string

print((package.searchpath("foo", "./?.lua")))
--> =nil

print((package.searchpath("testlib.foo", "./?.lua")))
--> =./testlib/foo.lua
