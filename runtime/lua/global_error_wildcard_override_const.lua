-- Test error: inner scope declares const, can't write even if outer has mutable wildcard

do
  global *
  x = 1
  do
    global<const> x  -- Redeclare x as const in inner scope
    x = 100          -- Error: x is const in this scope
    --> ~!!! parsing: .*attempt to assign to const global variable 'x'
  end
end
