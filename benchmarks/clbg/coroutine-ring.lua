-- The Computer Language Benchmarks Game
-- http://shootout.alioth.debian.org/
-- contributed by Sam Roberts
-- reviewed by Bruno Massa
--
-- From LuaJIT/LuaJIT-test-cleanup at 014708b (bench/), as a module: it
-- returns a function of n that returns the id of the last thread, in place of
-- writing it.  It closes the threads before returning: golua runs each
-- coroutine on a goroutine, which a suspended coroutine keeps until it is
-- closed, so each call would otherwise leave 503 goroutines behind.

return function(n)

-- fixed size pool
local poolsize  = 503
local threads   = {}

-- cache these to avoid global environment lookups
local create    = coroutine.create
local resume    = coroutine.resume
local yield     = coroutine.yield

local id        = 1
local token     = 0
local ok

local body = function(token)
  while true do
    token = yield(token + 1)
  end
end

-- create all threads
for id = 1, poolsize do
  threads[id] = create(body)
end

-- send the token
repeat
  if id == poolsize then
    id = 1
  else
    id = id + 1
  end
  ok, token = resume(threads[id], token)
until token == n

for id = 1, poolsize do
  coroutine.close(threads[id])
end

return id
end
