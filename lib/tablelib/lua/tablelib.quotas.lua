-- table.concat
local function mk(s, n)
    local t = {}
    for i = 1, n do
        t[i] = s
    end
    return t
end

do
    s1000 = ("a"):rep(1000)

    -- table.concat uses memory

    local ctx, res = runtime.callcontext({memlimit=5000}, table.concat, mk(s1000, 4))
    print(ctx, #res)
    --> =done	4000
    print(ctx.memused >= 4000)
    --> =true

    print(runtime.callcontext({memlimit=5000}, table.concat, mk(s1000, 5)))
    --> =killed

    -- table.concat uses cpu
    local ctx, res = runtime.callcontext({cpulimit=1000}, table.concat, mk("x", 100))
    print(ctx, #res)
    --> =done	100
    print(ctx.cpuused >= 100)
    --> =true

    print(runtime.callcontext({cpulimit=1000}, table.concat, mk("x", 1000)))
    --> =killed
end

-- table.insert
do
    -- cpu is consumed when inserting at the front
    local ctx = runtime.callcontext({cpulimit=1000}, table.insert, mk("x", 100), 1, "new")
    print(ctx)
    --> =done
    print(ctx.cpuused >= 100)
    --> =true

    print(runtime.callcontext({cpulimit=1000}, table.insert, mk("x", 1000), 1, "new"))
    --> =killed

    -- at the back, cpu doesn't depend on the size of the table
    local ctx1 = runtime.callcontext({cpulimit=1000}, table.insert, mk("x", 100), "new")
    print(ctx)
    --> =done
    print(ctx.cpuused >= 100)
    --> =true

    local ctx2 = runtime.callcontext({cpulimit=1000}, table.insert, mk("x", 1000), "new")
    print(ctx2)
    --> =done

    print(ctx2.cpuused / ctx1.cpuused < 1.2)
    --> =true
end

-- table.move
do
    -- table.move consumes memory

    local ctx = runtime.callcontext({memlimit=10000}, table.move, mk("x", 100), 1, 100, 101)
    print(ctx)
    --> =done

    print(runtime.callcontext({memlimit=10000}, table.move, mk("x", 1000), 1, 1000, 1001))
    --> =killed

    -- consumes less memory when ranges overlap
    ctx = runtime.callcontext({memlimit=10000}, table.move, mk("x", 1000), 1, 1000, 101)
    print(ctx)
    --> =done

    -- table.move consumes cpu

    local ctx = runtime.callcontext({cpulimit=1000}, table.move, mk("x", 100), 1, 100, 101)
    print(ctx)
    --> =done

    print(runtime.callcontext({cpulimit=1000}, table.move, mk("x", 1000), 1, 1000, 1001))
    --> =killed

    -- still consumes as much cpu when ranges overlap
    print(runtime.callcontext({cpulimit=1000}, table.move, mk("x", 1000), 1, 1000, 101))
    --> =killed
end

-- table.pack
do
    --table.pack consumes memory

    local ctx = runtime.callcontext({memlimit=10000}, table.pack, table.unpack(mk("x", 100)))
    print(ctx)
    --> =done

    print(runtime.callcontext({memlimit=10000}, table.pack, table.unpack(mk("x", 1000))))
    --> =killed

    --table.pack consumes cpu

    local ctx = runtime.callcontext({cpulimit=1000}, table.pack, table.unpack(mk("x", 100)))
    print(ctx)
    --> =done

    print(runtime.callcontext({cpulimit=1000}, table.pack, table.unpack(mk("x", 1000))))
    --> =killed
end

-- table.remove
do
    -- cpu is consumed when removing at the front
    local ctx = runtime.callcontext({cpulimit=1000}, table.remove, mk("x", 100), 1)
    print(ctx)
    --> =done
    print(ctx.cpuused >= 100)
    --> =true

    print(runtime.callcontext({cpulimit=1000}, table.remove, mk("x", 1000), 1))
    --> =killed

    -- at the back, cpu doesn't depend on the size of the table
    local ctx1 = runtime.callcontext({cpulimit=1000}, table.remove, mk("x", 100))
    print(ctx)
    --> =done
    print(ctx.cpuused >= 100)
    --> =true

    local ctx2 = runtime.callcontext({cpulimit=1000}, table.remove, mk("x", 1000))
    print(ctx2)
    --> =done

    print(ctx2.cpuused / ctx1.cpuused < 1.2)
    --> =true
end

-- table.sort
do
    -- table.sort consumes cpu
    local function unsorted(n)
        t = {}
        for i = 1, n do
            t[i] = n - i
        end
        return t
    end

    print(runtime.callcontext({cpulimit=1000}, table.sort, unsorted(10)))
    --> =done

    print(runtime.callcontext({cpulimit=1000}, table.sort, unsorted(100)))
    --> =killed
end