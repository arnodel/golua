local ctx, err = runtime.callcontext({fs={{prefix="testfiles/", denied="*"}}}, function()
    -- This operation is not allowed so an error is thrown
    io.open("testfiles/existing-file.txt")
end)
print(err)
--> ~.*safeio: operation not allowed

local ctx, err = runtime.callcontext({fs={{prefix="lua/testfiles/", allowed="r"}}}, function() 
    -- This operation is allowed but the file doesn't exist, so open returns an error
    local f <close>, err = io.open("lua/testfiles/non-existent-file")
    print(err == nil)
    --> =false

    -- This operation is not allowed so an error is thrown
    io.open("lua/testfiles/../otherdir/a-file")
end)
print(err)
--> ~.*safeio: operation not allowed

local ctx, err = runtime.callcontext({fs={{prefix="lua/testfiles/", allowed="*", denied="c"}}}, function() 
    -- This is allowed because the file exists
    local f <close>, err = io.open("lua/testfiles/existing-file.txt", "w")
    print(err)
    --> =nil

    -- This operation fails because file creation is disallowed
    io.open("lua/testfiles/non-existent-file", "w")
end)
print(err)
--> ~.*safeio: operation not allowed

local ctx, err = runtime.callcontext({fs={{prefix="lua/testfiles/", allowed="*"}}}, function() 
    -- This operation succeeds because file creation is allowed
    local f <close>, err = io.open("lua/testfiles/non-existent-file.txt", "w")
    print(err)
    --> =nil

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
