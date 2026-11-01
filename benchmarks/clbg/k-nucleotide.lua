-- The Computer Language Benchmarks Game
-- http://benchmarksgame.alioth.debian.org/
-- contributed by Mike Pall
--
-- From gligneul/Lua-Benchmarks at e1b3f24, as a module: it returns a
-- function of the input, the output of fasta, that returns the output as a
-- string.  write collects the output and nextline iterates over the lines
-- of the input, in place of io.write and io.lines().

local out, nout, nextline = {}, 0, nil

local function write(...)
  for i = 1, select("#", ...) do
    nout = nout + 1
    out[nout] = (select(i, ...))
  end
end

local function kfrequency(seq, freq, k, frame)
  local sub = string.sub
  local k1 = k - 1
  for i=frame,string.len(seq)-k1,k do
    local c = sub(seq, i, i+k1)
    freq[c] = freq[c] + 1
  end
end

local function freqdefault()
  return 0
end

local function count(seq, frag)
  local k = string.len(frag)
  local freq = setmetatable({}, { __index = freqdefault })
  for frame=1,k do kfrequency(seq, freq, k, frame) end
  write(freq[frag] or 0, "\t", frag, "\n")
end

local function frequency(seq, k)
  local freq = setmetatable({}, { __index = freqdefault })
  for frame=1,k do kfrequency(seq, freq, k, frame) end
  local sfreq, sn = {}, 1
  for c,v in pairs(freq) do sfreq[sn] = c; sn = sn + 1 end
  table.sort(sfreq, function(a, b)
    local fa, fb = freq[a], freq[b]
    return fa == fb and a > b or fa > fb
  end)
  sum = string.len(seq)-k+1
  for _,c in ipairs(sfreq) do
    write(string.format("%s %0.3f\n", c, (freq[c]*100)/sum))
  end
  write("\n")
end

local function readseq()
  local sub = string.sub
  for line in nextline do
    if sub(line, 1, 1) == ">" and sub(line, 2, 6) == "THREE" then break end
  end
  local lines, ln = {}, 0
  for line in nextline do
    local c = sub(line, 1, 1)
    if c == ">" then
      break
    elseif c ~= ";" then
      ln = ln + 1
      lines[ln] = line
    end
  end
  return string.upper(table.concat(lines, "", 1, ln))
end

return function(s)
  out, nout, nextline = {}, 0, s:gmatch("([^\n]*)\n")
  local seq = readseq()
  frequency(seq, 1)
  frequency(seq, 2)
  count(seq, "GGT")
  count(seq, "GGTA")
  count(seq, "GGTATT")
  count(seq, "GGTATTTTAATT")
  count(seq, "GGTATTTTAATTTATAGT")
  return table.concat(out, "", 1, nout)
end
