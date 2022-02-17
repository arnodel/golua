-- error in message handler, we eventually bail out
print(xpcall(error, error, "foo"))
--> =false	error in error handling


do
    -- The file doesn't exist but this is not an error!  This tests that the Go
    -- error is recognized as a *os.PathError and turned to a value.
    print(io.open("nosuchfile"))
    --> ~nil

    -- The file doesn't exist but this is not an error!  This tests that the Go
    -- error is recognized as a *os.LinkError and turned to a value.
    print(os.rename("nosuchfile", "newname"))
    --> ~nil
end

local a
 for i=1,'a' do 
 print(i) 
end
--> ~!!! runtime: error: luatest:\d+: 'for' limit: expected number, got string
