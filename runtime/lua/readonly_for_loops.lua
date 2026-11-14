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

-- The variable of a numeric for loop is read-only, and so is the first
-- variable of a generic for loop, its control variable: the loop passes it
-- back to the iterator.  The other variables of a generic for loop are
-- ordinary locals.

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

-- Test 3: The other variables of a for-in loop can be assigned to.  The
-- assignment lasts until the end of the iteration and changes nothing in
-- the table or in the iteration.
do
  local t = {10, 20, 30}
  local sum = 0
  for i, v in ipairs(t) do
    v = v + i
    sum = sum + v
  end
  print(sum, t[1], t[2], t[3])
end
--> =66	10	20	30

do
  local out = {}
  for i, a, b in function(_, i) if i < 3 then return i + 1, "a", nil end end, nil, 0 do
    b = b or "default"
    a, b = b, a
    out[#out + 1] = i .. a .. b
  end
  print(table.concat(out, " "))
end
--> =1defaulta 2defaulta 3defaulta

-- A closure can assign to them too, and each iteration has its own.
do
  local fs = {}
  for i, v in ipairs{"x", "y"} do
    fs[i] = function(s) v = v .. s return v end
  end
  print(fs[1]("1"), fs[2]("2"), fs[1]("3"))
end
--> =x1	y2	x13

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

-- Test 7: Several for-in variables: the first is const, whatever is done
-- with the others
test[[
  for a, b, c in string.gmatch("x y z", "%S+") do
    b, c = c, b
    a = "modified"
  end
]]
--> ~false\t.*attempt to reassign constant variable 'a'

test[[
  for a, b, c in string.gmatch("x y z", "%S+") do
    b, a = 1, 2
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
