local function checknumarg(f)
    local ok = not pcall(f) and not pcall(f, {})
    if ok then
        print "ok"
    else
        print "bad"
    end
end

do
    checknumarg(math.abs)
    --> =ok

    print(math.abs(12))
    --> =12

    print(math.abs(-12))
    --> =12

    print(math.abs(-0.5))
    --> =0.5

    print(math.abs("-123"))
    --> =123

    print(pcall(math.abs))
    --> ~false\t.*value needed

    print(pcall(math.abs, {}))
    --> ~false\t.*must be a number
end

do
    checknumarg(math.ceil)
    --> =ok

    print(math.ceil(123))
    --> =123

    print(math.ceil(12.7))
    --> =13

    print(math.ceil(-8.2))
    --> =-8

    print(math.ceil("888.5673"))
    --> =889
end

do
    checknumarg(math.exp)
    --> =ok

    print(math.exp(0))
    --> =1
    
end

do
    checknumarg(math.floor)
    --> =ok

    print(math.floor(123))
    --> =123

    print(math.floor(12.7))
    --> =12

    print(math.floor(-8.2))
    --> =-9

    print(math.floor("888.5673"))
    --> =888
end

do
    checknumarg(math.log)
    --> =ok

    print(math.log(1))
    --> =0
end

do
    print(math.max(3, 5, 2, 1))
    --> =5

    print(math.min(3, 5, 2, 1))
    --> =1
end

do
    checknumarg(math.modf)
    --> =ok

    print(math.modf(1.5))
    --> =1	0.5

    print(math.modf(-1.5))
    --> =-1	-0.5

    print(math.modf(1/0))
    --> =+Inf	0

    print(math.modf(-1/0))
    --> =-Inf	0

end

do
    checknumarg(math.rad)
    --> =ok

    checknumarg(math.deg)
    --> =ok

    print(math.rad(90) == math.pi / 2)
    --> =true

    print(math.deg(math.pi) == 180)
    --> =true
end

do
    checknumarg(math.randomseed)
    --> =ok

    local r = math.random()
    print(r >= 0 and r <= 1)
    --> =true

    local rands = {}
    math.randomseed(5)
    for i = 1, 20 do
        rands[i] = math.random(i, 2*i)
    end

    math.randomseed(5)
    local out, diff = 0, 0
    for i = 1, 20 do
        r = math.random(i, 2*i)
        if r < i or r > 2*i then
            out = out + 1
        end
        if rands[i] ~= r then
            diff = diff + 1
        end
    end
    print(out, diff)
    --> =0	0
end

do
    checknumarg(math.sin)
    --> =ok

    checknumarg(math.cos)
    --> =ok

    checknumarg(math.tan)
    --> =ok

    local function toz(f)
        return function(x)
            local y = f(x)
            if math.abs(y) < 1e-15 then
                return 0
            elseif math.abs(y) > 1e15 then
                return 1/0
            end
            return y
        end
    end
    local pi = math.pi
    local sin = toz(math.sin)
    local cos = toz(math.cos)
    local tan = toz(math.tan)

    print(sin(0), sin(pi/2), sin(pi), sin(3*pi/2))
    --> =0	1	0	-1

    print(cos(0), cos(pi/2), cos(pi), cos(3*pi/2))
    --> =1	0	-1	0

    print(tan(0), tan(pi/2), tan(pi), tan(3*pi/2))
    --> =0	+Inf	0	+Inf
end

do
    checknumarg(math.asin)
    --> =ok

    checknumarg(math.acos)
    --> =ok

    checknumarg(math.atan)
    --> =ok

    local function round(num) 
        if num >= 0 then return math.floor(num+.5) 
        else return math.ceil(num-.5) end
    end

    local function top(f)
        return function(x)
            local y = 4*f(x)/math.pi
            local ry = round(y)
            if math.abs(y - ry) > 1e-15 then
                return ry
            end
            return y
        end
    end

    local acos = top(math.acos)
    local asin = top(math.asin)
    local atan = top(math.atan)

    print(acos(-1), acos(0), acos(1))
    --> =4	2	0

    print(asin(-1), asin(0), asin(1))
    --> =-2	0	2

    print(atan(-1), atan(0), atan(1))
    --> =-1	0	1
end

do
    checknumarg(math.sqrt)
    --> =ok

    for i = 1, 5 do
        print(math.floor(1000*math.sqrt(i)))
    end
    --> =1000
    --> =1414
    --> =1732
    --> =2000
    --> =2236
end

do
    print(math.tointeger(56.0))
    --> =56

    print(math.tointeger("-123"))
    --> =-123

    print(math.tointeger({}))
    --> =nil

    print(math.tointeger(true))
    --> =nil
end

do
    print(math.type("123"))
    --> =nil

    print(math.type(123))
    --> =integer

    print(math.type(123.0))
    --> =float

    print(math.type(nil))
    --> =nil
end

do
    print(math.ult(1, 2))
    --> =true

    print(math.ult(-1, -2))
    --> =false

    print(math.ult(-1, 2))
    --> =false
end

do
    print(math.fmod(5, 2))
    --> =1

    -- TODO: fix implementation
    -- print(math.fmod(-5, 2))
    -- --> =-1
end
