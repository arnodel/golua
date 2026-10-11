-- Deeply nested source must fail to parse rather than exhaust the Go stack.
-- As in the reference implementation's errors.lua, 100 levels of each
-- construct parse and 500 do not.

local function testrep(init, rep, close, repc)
    local function gencode(n)
        return init .. string.rep(rep, n) .. close .. string.rep(repc, n)
    end
    local ok100 = load(gencode(100)) ~= nil
    local f, msg = load(gencode(500))
    return ok100, f == nil and string.find(msg, "too many syntax levels", 1, true) ~= nil
end

print(testrep("local a; a=", "{", "0", "}"))
--> =true	true
print(testrep("return ", "(", "2", ")"))
--> =true	true
print(testrep("local function a (x) return x end; return ", "a(", "2.2", ")"))
--> =true	true
print(testrep("", "do ", "", " end"))
--> =true	true
print(testrep("", "while a do ", "", " end"))
--> =true	true
print(testrep("local a; ", "if a then else ", "", " end"))
--> =true	true
print(testrep("", "function foo () ", "", " end"))
--> =true	true
print(testrep("local a = ''; return ", "a..", "'a'", ""))
--> =true	true
print(testrep("local a = 1; return ", "a^", "a", ""))
--> =true	true
print(testrep("local a = {}; return a", ".b", "", ""))
--> =true	true

-- The error is reported quickly however deep the input goes.
local f, msg = load(string.rep("(", 200000) .. "1" .. string.rep(")", 200000))
print(f, msg:match("too many syntax levels"))
--> =nil	too many syntax levels
