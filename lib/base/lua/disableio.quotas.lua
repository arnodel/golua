-- When IO is disabled, loadfile and dofile return errors

runtime.callcontext({io="off"}, function () 
    print(pcall(loadfile, "foo"))
    --> =false	io disabled

    print(pcall(dofile, "bar"))
    --> =false	io disabled
end)
