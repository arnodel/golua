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
