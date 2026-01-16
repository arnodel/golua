-- Test read-only for-loop variables (Lua 5.5 feature)
-- Uses load() to catch compile-time errors

local function test(src)
    local res, err = load(src)
    if res ~= nil then
        print(pcall(res))
    else
        print(false, err)
    end
end

-- Test 1: Numeric for-loop variable assignment should fail
test[[
  for i = 1, 10 do
    i = 5
  end
]]
--> ~false\t.*attempt to reassign constant variable 'i'

-- Test 2: For-in loop variable assignment should fail (first variable)
test[[
  local t = {a = 1, b = 2}
  for k, v in pairs(t) do
    k = "x"
  end
]]
--> ~false\t.*attempt to reassign constant variable 'k'

-- Test 3: For-in loop variable assignment should fail (second variable)
test[[
  local t = {a = 1, b = 2}
  for k, v in pairs(t) do
    v = 100
  end
]]
--> ~false\t.*attempt to reassign constant variable 'v'

-- Test 4: Reading loop variables is allowed
do
  local sum = 0
  for i = 1, 5 do
    sum = sum + i
  end
  print(sum)
end
--> =15

-- Test 5: Variables with same name outside loop are independent
do
  local i = 0
  for i = 1, 3 do
    -- loop i is const, but outer i is mutable
  end
  i = 99  -- OK: outer i is not const
  print(i)
end
--> =99

-- Test 6: Numeric for-loop with step - assignment should fail
test[[
  for i = 1, 10, 2 do
    i = i + 1
  end
]]
--> ~false\t.*attempt to reassign constant variable 'i'

-- Test 7: Multiple for-in variables, all are const
test[[
  for a, b, c in string.gmatch("x y z", "%S+") do
    a = "modified"
  end
]]
--> ~false\t.*attempt to reassign constant variable 'a'

-- Test 8: Nested for-loops, each has independent const variables
do
  for i = 1, 2 do
    for j = 1, 2 do
      print(i, j)
    end
  end
end
--> =1	1
--> =1	2
--> =2	1
--> =2	2

-- Test 9: Assignment to inner loop variable should fail
test[[
  for i = 1, 2 do
    for j = 1, 2 do
      j = 99
    end
  end
]]
--> ~false\t.*attempt to reassign constant variable 'j'

-- Test 10: For-in with single variable
test[[
  for line in io.lines() do
    line = "modified"
  end
]]
--> ~false\t.*attempt to reassign constant variable 'line'
