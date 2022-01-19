-- ok to jump over local dec. to end of block
x = 1
do
    goto l1
    local a = 23
    x = a
    ::l1::;
end

print(x)
--> =1

print(load([[
  repeat
    if x then goto cont end
    local xuxu = 10
    ::cont::
  until xuxu < x
]]))
--> ~nil\t.*no visible label 'cont'

-- Lua 5.4 forbids shadowing labels

print(load[[
  ::label::
  print"hello"
  ::label::
  print"bye"
]])
--> ~nil\t.*label 'label' already defined at line 1

print(load[[
  ::foo::
  do
    ::foo::
    goto foo
  end
]])
--> ~nil\t.*label 'foo' already defined at line 1

-- It's OK to reuse a label name that was defined outside the current function
-- scope.
do
  ::l2::
  local f = function(n)
    ::l2::
    print(n)
    n = n - 1
    if n > 0 then
      goto l2
    end
  end
  f(2)
end
--> =2
--> =1
