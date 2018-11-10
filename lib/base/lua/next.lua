print(pcall(next))
--> ~false\t.*value needed

print(pcall(next, 123))
--> ~false\t.*must be a table

print(pcall(next, {}, "abc"))
--> ~false\t.*invalid key
