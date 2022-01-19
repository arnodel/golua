-- error in message handler, we eventually bail out
print(xpcall(error, error, "foo"))
--> =false	error in error handling

local a
 for i=1,'a' do 
 print(i) 
end
--> =!!! runtime: error: luatest:6: expected numeric value

