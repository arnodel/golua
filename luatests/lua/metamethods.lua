local meta = {}
local function w(x)
    local t = {x=x}
    setmetatable(t, meta)
    return t
end

function meta.__tostring(x) return "<" .. x.x .. ">" end

function meta.__add(x, y) return w(x.x + y.x) end
function meta.__sub(x, y) return w(x.x - y.x) end
function meta.__mul(x, y) return w(x.x * y.x) end
function meta.__div(x, y) return w(x.x / y.x) end
function meta.__mod(x, y) return w(x.x % y.x) end
function meta.__pow(x, y) return w(x.x ^ y.x) end
function meta.__idiv(x, y) return w(x.x // y.x) end

function meta.__unm(x) return w(-x.x) end

function meta.__band(x, y) return w(x.x & y.x) end
function meta.__bor(x, y) return w(x.x | y.x) end
function meta.__bxor(x, y) return w(x.x ~ y.x) end
function meta.__shl(x, y) return w(x.x << y.x) end
function meta.__shr(x, y) return w(x.x >> y.x) end

function meta.__bnot(x) return w(~x.x) end

function meta.__eq(x, y) return x.x == y.x end
function meta.__lt(x, y) return x.x < y.x end
function meta.__le(x, y) return x.x <= y.x end

print(w(1))
--> =<1>

print(w(2)+w(3))
--> =<5>

print(w(10)*w(20))
--> =<200>

print((w(10) - w(2)) ^ w(2))
--> =<64>

print(w(51) // w(2) % w(7))
--> =<4>

print(-w(1))
--> =<-1>

print(w(1) << w(4) == w(2) << w(3))
--> =true

print(w(8) >> w(3) == w(1))
--> =true

print(w(~10) == ~w(10))
--> =true

print(w(8) | w(4))
--> =<12>

print(w(3) & w(5))
--> =<1>

print(w(10) ~ w(9))
--> =<3>

print(w(1) < w(2), w(3) <= w(3), w(4) > w(3), w(6) >= w(1))
--> =true	true	true	true

print(w(1) > w(2), w(3) >= w(3), w(4) < w(3), w(6) <= w(1))
--> =false	true	false	false
