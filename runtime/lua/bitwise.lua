print(-1 >> 56 == 0xff)
--> =true

print(1 >> -3)
--> =8

print(10 << -2)
--> =2

local function perr(s)
    local ok, err = pcall(load("return " .. s))
    print(err)
end

perr[[100 & "2"]]
--> ~.*attempt to perform bitwise and on a string value

perr[["100" | 23]]
--> ~.*attempt to perform bitwise or on a string value

perr[[~"55"]]
--> ~.*attempt to perform bitwise not on a string value

perr[[23 ~ "5"]]
--> ~.*attempt to perform bitwise xor on a string value

perr[["77" >> 9]]
--> ~.*attempt to perform bitwise shr on a string value

perr[["123" << "456" ]]
--> ~.*attempt to perform bitwise shl on a string value

perr[[0.5 | 1.2]]
--> ~.*number has no integer representation

perr[[45.2 >> "a"]]
--> ~.*attempt to perform bitwise shr on a string value