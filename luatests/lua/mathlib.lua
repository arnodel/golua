do
    print(math.abs(12))
    --> =12

    print(math.abs(-12))
    --> =12

    print(math.abs(-0.5))
    --> =0.5

    print(math.abs("-123"))
    --> =123
end

do
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
    print(math.exp(0))
    --> =1
    
end

do
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
    print(math.modf(1.5))
    --> =1	0.5

    print(math.modf(-1.5))
    --> =-1	-0.5
end

do
    print(math.rad(90) == math.pi / 2)
    --> =true

    print(math.deg(math.pi) == 180)
    --> =true
end

do
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
