-- Test error: writing to const global

do
  global<const> x = 42
  x = 100  -- Error: x is const
  --> ~!!! parsing: .*attempt to assign to const global variable 'x'
end
