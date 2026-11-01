-- The Computer Language Benchmarks Game
-- http://benchmarksgame.alioth.debian.org/
-- contributed by Jim Roseborough
-- modified by Victor Tang
-- optimized & replaced inefficient use of gsub with gmatch
-- partitioned sequence to prevent extraneous redundant string copy
--
-- From gligneul/Lua-Benchmarks at e1b3f24, as a module: it returns a
-- function of the input, the output of fasta, that returns the output as a
-- string, in place of reading stdin and writing stdout.

return function(input)
local out = {}
local function write(s) out[#out+1] = s end

seq = input
ilen, seq = #seq, seq:gsub('>[^%c]*%c*', ''):gsub('%c+', '')
clen = #seq

local variants = { 'agggtaaa|tttaccct',
                   '[cgt]gggtaaa|tttaccc[acg]',
                   'a[act]ggtaaa|tttacc[agt]t',
                   'ag[act]gtaaa|tttac[agt]ct',
                   'agg[act]taaa|ttta[agt]cct',
                   'aggg[acg]aaa|ttt[cgt]ccct',
                   'agggt[cgt]aa|tt[acg]accct',
                   'agggta[cgt]a|t[acg]taccct',
                   'agggtaa[cgt]|[acg]ttaccct', }

local subst = { B='(c|g|t)', D='(a|g|t)',   H='(a|c|t)', K='(g|t)',
                M='(a|c)',   N='(a|c|g|t)', R='(a|g)',   S='(c|g)',
                V='(a|c|g)', W='(a|t)',     Y='(c|t)' }

function countmatches(variant)
   local n = 0
   variant:gsub('([^|]+)|?', function(pattern)
      for _ in seq:gmatch(pattern) do n = n + 1 end
   end)
   return n
end

for _, p in ipairs(variants) do
   write( string.format('%s %d\n', p, countmatches(p)) )
end

function partitionstring(seq)
  local seg = math.floor( math.sqrt(#seq) )
  local seqtable = {}
  for nextstart = 1, #seq, seg do
    table.insert(seqtable, seq:sub(nextstart, nextstart + seg - 1))
  end
  return seqtable
end
function chunk_gsub(t, k, v)
  for i, p in ipairs(t) do
    t[i] = p:find(k) and p:gsub(k, v) or t[i]
  end
  return t
end

seq = partitionstring(seq)
for k, v in pairs(subst) do
  chunk_gsub(seq, k, v)
end
seq = table.concat(seq)
write(string.format('\n%d\n%d\n%d\n', ilen, clen, #seq))
return table.concat(out)
end

