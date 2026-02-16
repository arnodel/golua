-- Test valid global declarations (Lua 5.5 feature)

-- Test 1: Basic global declaration
do
  global print, x
  x = 42
  print("x =", x)
  --> =x =	42
end

-- Test 2: Global with const attribute and initialization
do
  global print
  global<const> PI = 3.14159
  print("PI =", PI)
  --> =PI =	3.14159
end

-- Test 3: Global wildcard (mutable)
do
  global *
  y = 100
  print("y =", y)
  --> =y =	100
end

-- Test 4: Nested scopes with explicit declaration overriding wildcard
do
  global print, z
  do
    global<const> *
    z = 200  -- Should work because z was declared mutable in outer scope
    print("z =", z)
    --> =z =	200
  end
end

-- Test 5: Multiple globals in one declaration
do
  global print, a, b, c
  a, b, c = 1, 2, 3
  print("a, b, c =", a, b, c)
  --> =a, b, c =	1	2	3
  a, b, c = nil, nil, nil  -- cleanup to avoid "already defined" error in later tests
end

-- Test 6: Global with value assignment
do
  global print, foo
  foo = "hello"
  print("foo =", foo)
  --> =foo =	hello
end

-- Test 7: Reading from const global
do
  global print
  global<const> CONST_VAL = 123
  local x = CONST_VAL
  print("x =", x)
  --> =x =	123
end

-- Test 8: Nested scopes - outer mutable, inner const wildcard
do
  global print, X, Y
  X = 10
  Y = 20
  do
    global<const> *
    -- X and Y can still be written because they were declared in outer scope
    X = 30
    Y = 40
    print("X, Y =", X, Y)
    --> =X, Y =	30	40
  end
  X, Y = nil, nil  -- cleanup to avoid "already defined" error in later tests
end

-- Test 9: Nested scopes - outer has wildcard, inner has explicit
do
  global *
  a = 1
  b = 2
  do
    global<const> c = 3
    -- a and b are still mutable from outer wildcard
    a = 10
    b = 20
    print("a, b, c =", a, b, c)
    --> =a, b, c =	10	20	3
  end
end

-- Test 10: Mix of const and mutable in same scope
do
  global print, mutable_var
  global<const> const_var = 100
  mutable_var = 50
  local sum = const_var + mutable_var
  print("sum =", sum)
  --> =sum =	150
end

-- Test 11: Triple nested scopes with different wildcard levels
do
  global print, A
  A = 1
  do
    global *
    B = 2
    do
      global<const> C = 3
      -- A is mutable (outer explicit), B is mutable (middle wildcard)
      A = 10
      B = 20
      print("A, B, C =", A, B, C)
      --> =A, B, C =	10	20	3
    end
  end
end

-- Test 12: Wildcard allows all undeclared names in that scope
do
  global *
  var1 = 1
  var2 = 2
  var3 = 3
  print("var1, var2, var3 =", var1, var2, var3)
  --> =var1, var2, var3 =	1	2	3
end

-- Test 13: Explicit const declaration, then reading multiple times
do
  global print
  global<const> READONLY = "constant"
  local x = READONLY
  local y = READONLY
  print("x, y =", x, y)
  --> =x, y =	constant	constant
end

-- Test 14: Nested with outer const wildcard, inner explicit mutable override
do
  global print
  do
    global<const> *
    do
      global M
      M = 123  -- M is mutable because explicitly declared in inner scope
      print("M =", M)
      --> =M =	123
    end
  end
end

-- Test 15: Multiple explicit declarations across nested scopes
do
  global print, x1
  x1 = 1
  do
    global x2
    x2 = 2
    do
      global x3
      x3 = 3
      -- All are accessible and mutable in their respective scopes
      x1 = 10
      x2 = 20
      x3 = 30
      print("x1, x2, x3 =", x1, x2, x3)
      --> =x1, x2, x3 =	10	20	30
    end
  end
end

-- Test 16: Const wildcard doesn't affect explicitly declared mutables in same scope
do
  global print, mutable
  global<const> *
  mutable = 999  -- This works - explicit mutable declaration
  print("mutable =", mutable)
  --> =mutable =	999
end

-- Test 17: Can redeclare mutable global as const
do
  global print, X
  X = 42
  global<const> X  -- Redeclare as const
  print("X after const redeclaration:", X)
  --> =X after const redeclaration:	42
end

-- Test 18: Global declaration shadows local from outer scope (GOLUA-003 fix)
do
  global print, assert
  local X = 10
  do
    global X
    X = 20  -- Should assign to global _ENV.X, not local X
  end
  assert(X == 10, "local X should still be 10")
  assert(_ENV.X == 20, "global X should be 20")
  print("Test 18 passed: global shadows local correctly")
  --> =Test 18 passed: global shadows local correctly
  _ENV.X = nil  -- cleanup
end

-- Test 19: Global with initialization reads local before shadowing
do
  global print, assert
  local Y = 100
  do
    global Y = Y  -- RHS Y reads local, then LHS Y becomes global
  end
  assert(Y == 100, "local Y should still be 100")
  assert(_ENV.Y == 100, "global Y should be 100 (from local)")
  print("Test 19 passed: global init reads local correctly")
  --> =Test 19 passed: global init reads local correctly
  _ENV.Y = nil  -- cleanup
end

-- Test 20: Reading global after it shadows local
do
  global print, assert
  local Z = 50
  _ENV.Z = 200
  do
    global Z
    local val = Z  -- Should read global Z (200), not local Z (50)
    assert(val == 200, "reading Z should get global value")
  end
  assert(Z == 50, "local Z should still be 50")
  print("Test 20 passed: reading shadowed global works")
  --> =Test 20 passed: reading shadowed global works
  _ENV.Z = nil  -- cleanup
end

-- Test 21: Strict mode propagates into nested functions - declared globals accessible (GOLUA-005 fix)
do
  global print, assert
  -- Once we declare any global explicitly, strict mode is enabled

  local function foo()
    -- print and assert are accessible because they were declared in outer scope
    return print ~= nil and assert ~= nil
  end
  assert(foo(), "declared globals should be accessible in nested function")
  print("Test 21 passed: declared globals accessible in nested function")
  --> =Test 21 passed: declared globals accessible in nested function
end

-- Test 22: Nested function can have its own global declarations to extend access
do
  global print, assert

  local function foo()
    -- This nested function declares its own global to extend access
    global tostring
    return tostring(42)
  end
  assert(foo() == "42", "nested function can declare its own globals")
  print("Test 22 passed: nested function can declare its own globals")
  --> =Test 22 passed: nested function can declare its own globals
end

-- Test 23: Global wildcard in outer scope propagates to nested functions
do
  global *

  local function foo()
    -- With global *, any global is accessible
    return print ~= nil and assert ~= nil and tostring ~= nil
  end
  assert(foo(), "global wildcard should allow access in nested function")
  print("Test 23 passed: global wildcard propagates to nested function")
  --> =Test 23 passed: global wildcard propagates to nested function
end

-- Test 24: Nested function with its own global declarations (strict mode)
do
  global print, assert
  global *  -- Outer has wildcard

  local function foo()
    global assert  -- Nested declares only assert, enabling its own strict mode
    -- Only assert should be accessible in the nested function now
    return assert ~= nil
  end
  assert(foo(), "nested function can have its own strict mode")
  print("Test 24 passed: nested function can have its own strict mode")
  --> =Test 24 passed: nested function can have its own strict mode
end

-- ============================================================================
-- Tests 25-35: Comprehensive nested function global declaration tests
-- ============================================================================

-- Test 25: Three levels - wildcard at top, no declarations in nested functions
do
  global *  -- Allow all globals
  local function level1()
    -- No declarations - inherits wildcard
    local function level2()
      -- No declarations - inherits wildcard
      local function level3()
        return print ~= nil and assert ~= nil and tostring ~= nil
      end
      return level3()
    end
    return level2()
  end
  assert(level1(), "three-level with wildcard allows all globals")
  print("Test 25 passed: three-level wildcard inheritance")
  --> =Test 25 passed: three-level wildcard inheritance
end

-- Test 26: Three levels - strict at top, legacy children inherit
do
  global print, assert, tostring
  local function level1()
    -- No declarations here - inherits from parent
    local function level2()
      -- No declarations here either
      local function level3()
        return print ~= nil and assert ~= nil and tostring ~= nil
      end
      return level3()
    end
    return level2()
  end
  assert(level1(), "strict mode should propagate through three levels")
  print("Test 26 passed: strict mode propagates through three levels")
  --> =Test 26 passed: strict mode propagates through three levels
end

-- Test 27: Three levels - wildcard at top, legacy children inherit
do
  global *
  local function level1()
    local function level2()
      local function level3()
        return print ~= nil and type ~= nil and pairs ~= nil
      end
      return level3()
    end
    return level2()
  end
  assert(level1(), "wildcard should propagate through three levels")
  print("Test 27 passed: wildcard propagates through three levels")
  --> =Test 27 passed: wildcard propagates through three levels
end

-- Test 28: Three levels - middle level adds restrictions
do
  global *  -- Level 0: allow all
  local function level1()
    global print, assert  -- Level 1: restrict to print, assert only
    local function level2()
      -- Level 2: inherits level1's restrictions
      return print ~= nil and assert ~= nil
    end
    return level2()
  end
  assert(level1(), "middle level can restrict parent's wildcard")
  print("Test 28 passed: middle level restricts parent's wildcard")
  --> =Test 28 passed: middle level restricts parent's wildcard
end

-- Test 29: Three levels - each level extends access
do
  global print, assert  -- Level 0: print and assert (need assert for the test)
  local function level1()
    global tostring  -- Level 1: adds tostring
    local function level2()
      global type  -- Level 2: adds type
      return print ~= nil and assert ~= nil and tostring ~= nil and type ~= nil
    end
    return level2()
  end
  assert(level1(), "each level can extend global access")
  print("Test 29 passed: each level extends global access")
  --> =Test 29 passed: each level extends global access
end

-- Test 30: Strict parent, child with wildcard extends access
do
  global print, assert
  local function foo()
    global *  -- Child uses wildcard to extend access
    return print ~= nil and assert ~= nil and tostring ~= nil and type ~= nil
  end
  assert(foo(), "child wildcard extends parent's strict declarations")
  print("Test 30 passed: child wildcard extends strict parent")
  --> =Test 30 passed: child wildcard extends strict parent
end

-- Test 31: Legacy parent, strict child restricts
do
  -- No global declarations here (legacy mode)
  local function foo()
    global print, assert  -- Child restricts to just these
    return print ~= nil and assert ~= nil
  end
  assert(foo(), "strict child in legacy parent")
  print("Test 31 passed: strict child in legacy parent")
  --> =Test 31 passed: strict child in legacy parent
end

-- Test 32: Const wildcard at parent, mutable declaration at child
do
  global print, assert
  global<const> *  -- Parent: all undeclared globals are const
  local function foo()
    global tostring  -- Child: tostring is mutable (explicit declaration)
    return tostring ~= nil
  end
  assert(foo(), "child can declare mutable over parent's const wildcard")
  print("Test 32 passed: mutable declaration over const wildcard")
  --> =Test 32 passed: mutable declaration over const wildcard
end

-- Test 33: Three levels with mixed modes
do
  global print, assert  -- Level 0: strict with print, assert
  local function level1()
    global *  -- Level 1: wildcard (extends to allow all)
    local function level2()
      global tostring  -- Level 2: strict again (only tostring + inherited)
      -- Should have access to: print, assert (from L0), tostring (from L2)
      -- But L2 has its own strict mode, so only tostring is "declared" at L2
      -- However, print and assert are declared at L0, so should still be accessible
      return print ~= nil and assert ~= nil and tostring ~= nil
    end
    return level2()
  end
  assert(level1(), "three levels with alternating modes")
  print("Test 33 passed: three levels with alternating modes")
  --> =Test 33 passed: three levels with alternating modes
end

-- Test 34: Deeply nested - 4 levels
do
  global print, assert
  local function a()
    local function b()
      local function c()
        local function d()
          return print ~= nil and assert ~= nil
        end
        return d()
      end
      return c()
    end
    return b()
  end
  assert(a(), "four-level nesting with inherited strict mode")
  print("Test 34 passed: four-level nesting")
  --> =Test 34 passed: four-level nesting
end

-- Test 35: Sibling functions with different declarations
do
  global print, assert

  local function foo()
    global tostring  -- foo extends with tostring
    return tostring(42)
  end

  local function bar()
    global type  -- bar extends with type (different from foo)
    return type(42)
  end

  assert(foo() == "42", "foo should have tostring")
  assert(bar() == "number", "bar should have type")
  print("Test 35 passed: sibling functions with different extensions")
  --> =Test 35 passed: sibling functions with different extensions
end

global print
print("All tests passed!")
--> =All tests passed!
