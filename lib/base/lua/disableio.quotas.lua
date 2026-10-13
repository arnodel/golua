-- When IO is disabled, loadfile and dofile return errors

runtime.callcontext({flags="iosafe"}, function () 
    print(pcall(loadfile, "foo"))
    --> ~false\t.*: safeio: operation not allowed

    print(pcall(dofile, "bar"))
    --> ~false\t.*: safeio: operation not allowed
end)

-- But they work on files in directories granted by the fs quota

print(runtime.callcontext({fs={{dir="lua", allowed="r"}}}, function ()
    print(loadfile("lua/print.lua") ~= nil)
    --> =true

    print(pcall(loadfile, "base.go"))
    --> ~false\t.*: safeio: operation not allowed
end))
--> =done
