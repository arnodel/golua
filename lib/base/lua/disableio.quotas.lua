-- When IO is disabled, loadfile and dofile return errors

runtime.callcontext({flags="iosafe"}, function () 
    print(pcall(loadfile, "foo"))
    --> =false	missing flags: iosafe

    print(pcall(dofile, "bar"))
    --> =false	missing flags: iosafe
end)
