-- Test that redefining a global with a value errors at runtime (Lua 5.5)
-- The error "global 'name' already defined" is raised when _ENV[name] is non-nil
-- at the point of a "global name = value" declaration.

-- Basic redefinition error
do
  global pcall, print
  global x = 5
  print(pcall(function() global x = 10 end))
  --> ~false.*global 'x' already defined
  _ENV.x = nil
end

-- Re-declaration without value is OK
do
  global print
  global x = 5
  global x  -- no value, no error
  print("redecl ok", x)
  --> =redecl ok	5
  _ENV.x = nil
end

-- Defining after setting to nil suppresses the error
do
  global pcall, print
  global x = 5
  print(pcall(function() global x = 10 end))
  --> ~false.*global 'x' already defined
  x = nil
  global x = 10  -- OK because _ENV.x is now nil
  print("after nil", x)
  --> =after nil	10
  _ENV.x = nil
end

-- Standard library globals can't be redefined
do
  global pcall, print
  print(pcall(function() global print = 10 end))
  --> ~false.*global 'print' already defined
end

-- Implicit global then explicit define errors
do
  global *
  foo = 5
  print(pcall(function() global foo = 10 end))
  --> ~false.*global 'foo' already defined
  _ENV.foo = nil
end

-- Global function redefinition errors
do
  global pcall, print
  global function bar() return 1 end
  print(pcall(function() global function bar() return 2 end end))
  --> ~false.*global 'bar' already defined
  _ENV.bar = nil
end

-- Multiple globals with first already defined
do
  global pcall, print
  global a = 1
  print(pcall(function() global a, b = 10, 20 end))
  --> ~false.*global 'a' already defined
  _ENV.a = nil
  _ENV.b = nil
end

-- false is non-nil, so redefinition errors
do
  global pcall, print
  global x = false
  print(pcall(function() global x = true end))
  --> ~false.*global 'x' already defined
  _ENV.x = nil
end

-- local _ENV redefinition: check uses the correct environment
do
  global pcall, print, load
  local f = load("local _ENV = {AA = false}; global AA = 10")
  print(pcall(f))
  --> ~false.*global 'AA' already defined
end
