-- The Computer Language Benchmarks Game
-- http://benchmarksgame.alioth.debian.org/
-- contributed by Mike Pall
--
-- From gligneul/Lua-Benchmarks at e1b3f24, as a module: it returns a
-- function of the maximum depth that returns the checks' sum, in place of
-- writing them.

local function BottomUpTree(item, depth)
  if depth > 0 then
    local i = item + item
    depth = depth - 1
    local left, right = BottomUpTree(i-1, depth), BottomUpTree(i, depth)
    return { item, left, right }
  else
    return { item }
  end
end

local function ItemCheck(tree)
  if tree[2] then
    return tree[1] + ItemCheck(tree[2]) - ItemCheck(tree[3])
  else
    return tree[1]
  end
end

return function(N)
  local sum = 0
  local mindepth = 4
  local maxdepth = mindepth + 2
  if maxdepth < N then maxdepth = N end

  do
    local stretchdepth = maxdepth + 1
    local stretchtree = BottomUpTree(0, stretchdepth)
    sum = sum + ItemCheck(stretchtree)
  end

  local longlivedtree = BottomUpTree(0, maxdepth)

  for depth=mindepth,maxdepth,2 do
    local iterations = 2 ^ (maxdepth - depth + mindepth)
    local check = 0
    for i=1,iterations do
      check = check + ItemCheck(BottomUpTree(1, depth)) +
              ItemCheck(BottomUpTree(-1, depth))
    end
    sum = sum + iterations*2 + check
  end

  return sum + ItemCheck(longlivedtree)
end
