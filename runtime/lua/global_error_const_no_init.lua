-- Test error: const global declared without initialization, then assigned

do
  global<const> x
  x = 42  -- Error: x is const, can't assign even though it was just declared
  --> ~!!! parsing: .*attempt to assign to const global variable 'x'
end
