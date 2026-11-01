#!/bin/sh
# Layout-averaged A/B comparison of golua commits.
#
#   benchmarks/layout_ab.sh [-r ROUNDS] [-b REGEX] [-w WORKDIR] NAME=REV NAME=REV ...
#
# REGEX is passed to -test.bench, which matches each level of a benchmark's
# name separately: 'Standard/md5|life/impl=golua' selects two benchmarks.
#
# The speed of the run loop (LuaCont.RunInThread) depends on where its code
# lands in memory, not only on what it does: moving the same code by 16 bytes
# can change a benchmark by 1 to 3%.  To keep that out of comparisons, each
# revision is built four times, with a padding function inserted just before
# RunInThread so that the loop's entry (the "opcode := opcodes[pc]" line)
# lands at each offset 0, 16, 32 and 48 within a 64-byte line.  The
# benchmarks then run interleaved over all the builds, and each revision's
# results from its four builds are pooled, so the comparison is between
# averages over layouts.
#
# Builds use the noquotas tag.  The first NAME=REV is the baseline.  Needs
# benchstat on the path.
set -e

rounds=3
regex=.
work=
while getopts r:b:w: opt; do
	case $opt in
	r) rounds=$OPTARG ;;
	b) regex=$OPTARG ;;
	w) work=$OPTARG ;;
	*) exit 2 ;;
	esac
done
shift $((OPTIND - 1))
[ $# -ge 2 ] || { echo "usage: $0 [-r ROUNDS] [-b REGEX] [-w WORKDIR] NAME=REV NAME=REV ..." >&2; exit 2; }

repo=$(git rev-parse --show-toplevel)
[ -n "$work" ] || work=$(mktemp -d)
mkdir -p "$work"
echo "work directory: $work" >&2

# loop_offset BINARY ENTRYLINE prints the offset of the loop entry in its
# 64-byte line.
loop_offset() {
	go tool objdump -s 'runtime\.\(\*LuaCont\)\.RunInThread$' "$1" 2>/dev/null |
		awk -v l="luacont.go:$2" '$1 == l { print $2; exit }' |
		{ read -r addr && echo $((addr % 64)); }
}

# build NAME REV builds REV at the four offsets as $work/NAME-OFFSET.test.
build() {
	name=$1
	rev=$2
	src="$work/src-$name"
	rm -rf "${src:?}"
	mkdir -p "$src"
	git -C "$repo" archive "$rev" | tar -x -C "$src"
	file="$src/runtime/luacont.go"
	# The padding functions go just before RunInThread (functions are laid
	# out in source order).  layoutPad is kept reachable through an init, or
	# the linker would drop it.
	python3 - "$file" <<'EOF'
import sys
p = sys.argv[1]
s = open(p).read()
mark = "// RunInThread implements Cont.RunInThread.\n"
assert s.count(mark) == 1, "RunInThread marker not found"
pad = """// padStep and layoutPad only shift the address of the code that follows
// them.  Calls cannot be folded away, so each statement of layoutPad adds a
// few instructions.
//
//go:noinline
func padStep(x int) int { return x + 1 }

//go:noinline
func layoutPad(x int) int {
	// PAD
	return x
}

var padSink func(int) int

func init() { padSink = layoutPad }

"""
open(p, "w").write(s.replace(mark, pad + mark))
EOF
	found=""
	k=0
	while [ $k -le 40 ]; do
		python3 - "$file" "$k" <<'EOF'
import sys
p, k = sys.argv[1], int(sys.argv[2])
s = open(p).read()
a = s.index("\t// PAD\n")
b = s.index("\treturn x\n", a)
body = "\tx = padStep(x)\n" * k
open(p, "w").write(s[:a] + "\t// PAD\n" + body + s[b:])
EOF
		# The padding moves the loop entry's line number too.
		entry=$(grep -n 'opcode := opcodes\[pc\]' "$file" | head -1 | cut -d: -f1)
		bin="$work/$name-try.test"
		(cd "$src" && go test -c -tags noquotas -o "$bin" ./benchmarks >/dev/null)
		off=$(loop_offset "$bin" "$entry")
		case " $found " in
		*" $off "*) rm -f "$bin" ;;
		*)
			mv "$bin" "$work/$name-$off.test"
			found="$found $off"
			echo "$name: loop entry at offset $off with $k padding statements" >&2
			;;
		esac
		[ $(echo $found | wc -w) -eq 4 ] && return 0
		k=$((k + 1))
	done
	echo "$name: could not reach all four offsets (got$found)" >&2
	exit 1
}

for spec in "$@"; do
	build "${spec%%=*}" "${spec#*=}"
done

cd "$repo/benchmarks"
for spec in "$@"; do : >"$work/${spec%%=*}.txt"; done
for round in $(seq 1 "$rounds"); do
	for off in 0 16 32 48; do
		for spec in "$@"; do
			name=${spec%%=*}
			"$work/$name-$off.test" -test.run '^$' -test.bench "$regex" -test.count 1 2>&1 |
				grep '^Benchmark' >>"$work/$name.txt"
		done
	done
done

# A pattern that matches no benchmark runs nothing, quietly.
for spec in "$@"; do
	[ -s "$work/${spec%%=*}.txt" ] || { echo "no benchmark matched $regex" >&2; exit 1; }
done

first=${1%%=*}
for spec in "$@"; do
	name=${spec%%=*}
	[ "$name" = "$first" ] && continue
	echo "=== $first vs $name (pooled over 4 layouts x $rounds rounds) ==="
	benchstat "$work/$first.txt" "$work/$name.txt" | sed -E 's/ {3,}/   /g' | grep -E '^Standard|geomean'
done
