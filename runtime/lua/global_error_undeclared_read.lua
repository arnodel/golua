-- Test error: reading from undeclared global when globals are declared

do
  global print, x
  print(y)  -- Error: y is not declared
  --> ~!!! parsing: .*attempt to read undeclared global variable 'y'
end
