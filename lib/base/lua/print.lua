_tostring = tostring
function tostring(x)
    if x == 42 then
        error("42!!", 0)
    elseif x == 43 then
        return 43
    end
    return _tostring(x)
end

print(pcall(print, 42))
--> =false	42!!

print(pcall(print, 43))
--> ~false\t.*: tostring must return a string

tostring = _tostring
