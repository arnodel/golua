-- Test error: writing to undeclared global when globals are declared

do
  global x
  y = 100  -- Error: y is not declared
  --> ~!!! parsing: .*attempt to assign to undeclared global variable 'y'
end
