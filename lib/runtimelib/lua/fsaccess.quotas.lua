-- Tests for fs access.  The 'fs' field in the context is a list of rule, each rule of the form
-- { prefix='prefix/path', allowed=ACTION_STRING, denied=ACTION_STRING }
-- An ACTION_STRING is a string made of the following characters:
-- - '*': all actions
-- - 'r': read a file
-- - 'w': write a file
-- - 'c': create a file
-- - 'd': delete a file
-- - 'C': create any file in a directory (weird, used for creating temp files)

-- Deny all operations in lua/testfiles/
local ctx, err = runtime.callcontext({fs={{prefix="testfiles/", denied="*"}}}, function()
    -- This operation is not allowed so an error is thrown
    io.open("testfiles/existing-file.txt")
end)
print(err)
--> ~.*safeio: operation not allowed

-- Allow only reading files in lua/testfiles/
local ctx, err = runtime.callcontext({fs={{prefix="lua/testfiles/", allowed="r"}}}, function() 
    -- This operation is allowed but the file doesn't exist, so open returns an error
    local f, err = io.open("lua/testfiles/non-existent-file")
    print(err == nil)
    --> =false

    -- This operation is not allowed so an error is thrown
    io.open("lua/testfiles/../otherdir/a-file")
end)
print(err)
--> ~.*safeio: operation not allowed

-- Allow all operations apart from creating files in lua/testfiles/
local ctx, err = runtime.callcontext({fs={{prefix="lua/testfiles/", allowed="*", denied="c"}}}, function() 
    -- This is allowed because the file exists
    local f, err = io.open("lua/testfiles/existing-file.txt", "w")
    print(err)
    --> =nil

    f:close()

    -- This operation fails because file creation is disallowed
    io.open("lua/testfiles/non-existent-file", "w")
end)
print(err)
--> ~.*safeio: operation not allowed

-- Allow all operations in lua/testfiles/
local ctx, err = runtime.callcontext({fs={{prefix="lua/testfiles/", allowed="*"}}}, function() 
    -- This operation succeeds because file creation is allowed
    local f, err = io.open("lua/testfiles/non-existent-file.txt", "w")
    print(err)
    --> =nil

    f:close()

    -- The file can be renamed within lua/testfiles
    print(os.rename("lua/testfiles/non-existent-file.txt", "lua/testfiles/renamed.txt"))
    --> =true

    -- The file cannot be moved outside of lua/testfiles
    print(pcall(os.rename, "lua/testfiles/renamed.txt", "lua/otherdir/moved.txt"))
    --> ~false\t.*safeio: operation not allowed

    -- The file can be deleted
    print(os.remove("lua/testfiles/renamed.txt"))
    --> =true
end)
