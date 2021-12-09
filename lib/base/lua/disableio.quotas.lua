-- When IO is disabled, loadfile and dofile return errors

runtime.callcontext({flags="iosafe"}, function () 
    print(pcall(loadfile, "foo"))
    --> ~false\t.*: missing flags: iosafe

    print(pcall(dofile, "bar"))
    --> ~false\t.*: missing flags: iosafe
end)
