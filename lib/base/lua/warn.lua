-- off by default
warn("discarded warning")

warn("@on")

warn("a warning")
--> =Test warning: a warning

warn("this", 12, 9.5)
--> =Test warning: this129.5

warn("some", "@on", "other", "@off")
--> =Test warning: some@onother@off

warn("@off")

warn("discarded")
print("hello")
--> =hello

print(pcall(warn))
--> ~false\t.*value needed

print(pcall(warn, {}))
--> ~false\t.*string expected

