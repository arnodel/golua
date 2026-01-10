-- Test error: writing when const wildcard is in effect

do
  global<const> *
  x = 42  -- Error: all globals are const due to wildcard
  --> ~!!! parsing: .*attempt to assign to const global variable 'x'
end
