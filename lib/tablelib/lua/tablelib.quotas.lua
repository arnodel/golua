-- table.concat
do
    s1000 = ("a"):rep(1000)

    local function mk(s, n)
        local t = {}
        for i = 1, n do
            t[i] = s
        end
        return t
    end

    -- table.concat uses memory

    local status, res = runtime.callcontext({memlimit=5000}, table.concat, mk(s1000, 4))
    print(status, #res)
    --> =done	4000
    print(status.memused >= 4000)
    --> =true

    print(runtime.callcontext({memlimit=5000}, table.concat, mk(s1000, 5)))
    --> =killed

    -- table.concat uses cpu
    local status, res = runtime.callcontext({cpulimit=1000}, table.concat, mk("x", 100))
    print(status, #res)
    --> =done	100
    print(status.cpuused >= 100)
    --> =true

    print(runtime.callcontext({cpulimit=1000}, table.concat, mk("x", 1000)))
    --> =killed
end
