-- Test error: Cannot assign after redeclaring as const

do
  global X
  X = 42
  global<const> X  -- Redeclare as const
  X = 100          -- Error: X is now const
  --> ~!!! parsing: .*attempt to assign to const global variable 'X'
end
