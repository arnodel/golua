-- Test global function declarations (Lua 5.5 feature)

-- Test 1: Basic global function declaration
do
  global print
  global function greet()
    print("Hello from global function!")
  end
  greet()
  --> =Hello from global function!
end

-- Test 2: Global function with parameters
do
  global print
  global function add(a, b)
    return a + b
  end
  print("add(2, 3) =", add(2, 3))
  --> =add(2, 3) =	5
end

-- Test 3: Global function with multiple return values
do
  global print
  global function swap(a, b)
    return b, a
  end
  local x, y = swap(1, 2)
  print("swap(1, 2) =", x, y)
  --> =swap(1, 2) =	2	1
end

-- Test 4: Global function can call other global functions
do
  global print
  global function double(x)
    return x * 2
  end
  global function quadruple(x)
    return double(double(x))
  end
  print("quadruple(5) =", quadruple(5))
  --> =quadruple(5) =	20
end

-- Test 5: Global function with closure over locals
do
  global print
  local multiplier = 10
  global function multiplyBy10(x)
    return x * multiplier
  end
  print("multiplyBy10(7) =", multiplyBy10(7))
  --> =multiplyBy10(7) =	70
end

-- Test 6: Global function with varargs
do
  global print, table, ipairs
  global function printAll(...)
    local args = {...}
    for i, v in ipairs(args) do
      print("arg", i, "=", v)
    end
  end
  printAll("a", "b", "c")
  --> =arg	1	=	a
  --> =arg	2	=	b
  --> =arg	3	=	c
end

-- Test 7: Recursive global function
do
  global print
  global function factorial(n)
    if n <= 1 then
      return 1
    else
      return n * factorial(n - 1)
    end
  end
  print("factorial(5) =", factorial(5))
  --> =factorial(5) =	120
end

-- Test 8: Global function can be reassigned (not const)
do
  global print
  global function foo()
    return "original"
  end
  print("foo() =", foo())
  --> =foo() =	original

  foo = function()
    return "reassigned"
  end
  print("foo() after reassign =", foo())
  --> =foo() after reassign =	reassigned
end

-- Test 9: Global function declaration is scope-limited (like local)
-- The function value is assigned to _ENV, but the declaration must be in scope
do
  global print, innerFunc
  do
    global function innerFunc()
      return "from inner scope"
    end
    print("innerFunc() inside =", innerFunc())
    --> =innerFunc() inside =	from inner scope
  end
  -- Function is accessible because innerFunc was declared in outer scope
  print("innerFunc() outside =", innerFunc())
  --> =innerFunc() outside =	from inner scope
end

-- Test 10: Multiple global functions in sequence
do
  global print
  global function f1() return 1 end
  global function f2() return 2 end
  global function f3() return 3 end
  print("f1, f2, f3 =", f1(), f2(), f3())
  --> =f1, f2, f3 =	1	2	3
end

-- Test 11: Global function with wildcard mode active
do
  global *
  global function helper()
    return "helped"
  end
  print("helper() =", helper())
  --> =helper() =	helped
end

-- Test 12: Global function that returns a function
do
  global print
  global function makeAdder(n)
    return function(x)
      return x + n
    end
  end
  local add5 = makeAdder(5)
  print("add5(10) =", add5(10))
  --> =add5(10) =	15
end

-- Test 13: Global function shadows local variable (GOLUA-006 fix)
do
  global print, assert
  local foo = 20  -- local variable
  do
    global function foo(x)
      if x == 0 then return 1 else return 2 * foo(x - 1) end
    end
    -- Inside this block, foo refers to the global function
    assert(foo == _ENV.foo, "foo should be global inside block")
    assert(foo(4) == 16, "foo(4) should be 16")
  end
  -- Outside the block, foo refers to the local variable
  assert(_ENV.foo(4) == 16, "_ENV.foo(4) should be 16")
  assert(foo == 20, "local foo should still be 20")
  print("Test 13 passed: global function shadows local correctly")
  --> =Test 13 passed: global function shadows local correctly
  _ENV.foo = nil  -- cleanup
end

-- Test 14: Local function shadows outer global declaration
do
  global print, assert
  global fact = false  -- global set to false
  do
    local res = 1
    local function fact(n)
      if n == 0 then return res else return n * fact(n - 1) end
    end
    -- Inside this block, fact is the local recursive function
    assert(fact(5) == 120, "local fact(5) should be 120")
  end
  -- Outside, fact is still the global (false)
  assert(fact == false, "global fact should still be false")
  print("Test 14 passed: local function shadows global correctly")
  --> =Test 14 passed: local function shadows global correctly
  _ENV.fact = nil  -- cleanup
end

global print
print("All global function tests passed!")
--> =All global function tests passed!
