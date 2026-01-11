-- Test named vararg tables (Lua 5.5 feature)

local function test(src)
    local res, err = load(src)
    if res ~= nil then
        print(pcall(res))
    else
        print(false, err)
    end
end

-- Test 1: Named vararg basic access
do
  local function f(...args)
    print(args[1], args[2], args[3])
  end
  f(10, 20, 30)
end
--> =10	20	30

-- Test 2: Named vararg with regular parameters
do
  local function f(a, b, ...rest)
    print(a, b, rest[1], rest[2])
  end
  f(1, 2, 3, 4, 5)
end
--> =1	2	3	4

-- Test 3: Named vararg is read-only
test[[
  local function f(...args)
    args = {}
  end
  f(1)
]]
--> ~false\t.*attempt to reassign constant variable 'args'

-- Test 4: Both ... and name work
do
  local function f(...args)
    print(...)
    print(args[1], args[2])
  end
  f(10, 20, 30)
end
--> =10	20	30
--> =10	20

-- Test 5: Backward compatibility - vararg without name
do
  local function f(...)
    local t = {...}
    print(t[1], t[2])
  end
  f(100, 200)
end
--> =100	200

-- Test 6: Empty varargs
do
  local function f(...args)
    print(args[1])
  end
  f()
end
--> =nil

-- Test 7: Vararg in nested function
do
  local function outer(...x)
    local function inner()
      return x[1], x[2]
    end
    print(inner())
  end
  outer(5, 10, 15)
end
--> =5	10

-- Test 8: Named vararg with select
do
  local function f(...args)
    print(select('#', ...))
    print(args[2])
  end
  f(1, 2, 3)
end
--> =3
--> =2

-- Test 9: Can modify table elements (table is mutable, var is const)
do
  local function f(...args)
    args[1] = 999
    print(args[1], args[2])
  end
  f(10, 20)
end
--> =999	20

-- Test 10: Named vararg with iteration
do
  local function sum(...nums)
    local total = 0
    for i = 1, select('#', ...) do
      total = total + nums[i]
    end
    print(total)
  end
  sum(1, 2, 3, 4, 5)
end
--> =15

-- Test 11: Shared mutation - modify args affects ...
do
  local function f(...args)
    print(...)
    args[1] = 999
    print(...)
  end
  f(10, 20, 30)
end
--> =10	20	30
--> =999	20	30

-- Test 12: Shared mutation - setting nil
do
  local function f(...args)
    print(...)
    args[2] = nil
    print(...)
  end
  f(10, 20, 30)
end
--> =10	20	30
--> =10	nil	30
