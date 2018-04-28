local f = load("print('hello')")
f()
--> =hello

do
    local env = {x = 1}
    local f = load("x = 2", "chunk", "bt", env)
    print(env.x)
--> =1
    f()
    print(env.x)
--> =2
end

load("print(...)")(1, 2)
--> =1	2
