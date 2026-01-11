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

global print
print("All tests passed!")
--> =All tests passed!
