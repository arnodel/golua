local ctx, err = runtime.callcontext({fs={{prefix="testfiles/", denied="*"}}}, function()
    -- This operation is not allowed so an error is thrown
    io.open("testfiles/file.txt")
end)
print(err)
--> ~.*safeio: operation not allowed

local ctx, err = runtime.callcontext({fs={{prefix="testfiles/", allowed="r"}}}, function() 
    -- This operation is allowed but the file doesn't exist, so open returns an error
    f, err = io.open("testfiles/non-existent-file")
    print(err == nil)
    --> =false

    -- This operation is not allowed so an error is thrown
    io.open("testfiles/../otherdir/a-file")
end)
print(err)
--> ~.*safeio: operation not allowed

local ctx, err = runtime.callcontext({fs={{prefix="testfiles/", allowed="*", denied="c"}}}, function() 
    -- This operation is not allowed because file creation is disallowed
    io.open("testfiles/non-existent-file", "w")
end)
print(err)
--> ~.*safeio: operation not allowed
