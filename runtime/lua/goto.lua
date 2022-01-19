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
--> ~nil\t.*undefined label 'cont'

-- (bugfix) A return statement prevents back labels
print(load[[
  goto L
  local a = 1
  ::L:: return a
]])
--> ~nil\t.*undefined label 'L'
