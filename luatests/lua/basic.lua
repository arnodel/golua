print("hello, world!")
--> =hello, world!

print(1+2)
--> =3

print(1 == 1.0)
--> =true

print(1, 2)
--> =1	2

print("hello," .. " " .. "world!")
--> =hello, world!

local function max(x, y)
  if x > y then
    return x
  end
  return y
end
print(max(2, 3), max(3, 2))
--> =3	3

local function sum(n)
    local s = 0
    for i = 1,n do
        s = s + i
    end
    return s
end
print(sum(10))
--> =55

local function fac(n)
  if n == 0 then
    return 1
  end
  return n * fac(n-1)
end
print(fac(10))
--> =3628800

local function twice(f)
  return function(x)
    return f(f(x))
  end
end
local function square(x)
  return x*x
end
print(twice(square)(2)) -- (2 ^ 2) ^ 2
--> =16

local function p(...)
  print(">>>", ...)
end
p(1, 2, 3)
--> =>>>	1	2	3

local function g(x)
  error(x .. "ld!")
end
local function f(x)
    g(x .. ", wor")
end
print(pcall(f, "hello"))
--> =false	hello, world!

local function f()
    return 1, 2
end
local x, y
x, y = f()
print(x + y)
--> =3

error("hello")
--> ~^!!! runtime:

