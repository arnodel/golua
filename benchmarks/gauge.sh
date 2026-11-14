#!/bin/sh
# Time golua at several revisions against C Lua, on the benchmarks of the
# working tree.
#
#   benchmarks/gauge.sh [-n rounds] [-l lua] REV...
#
# Each revision's interpreter is built with this checkout's benchmarks/ in
# place of its own, so every revision runs the same programs, with
# -tags noquotas, the build performance work targets, and with the revision's
# own profile (default.pgo) if it has one, as a build of its golua command
# would be.  The revisions run in turn, round after round, so that a machine
# getting slower or faster affects them all alike; C Lua runs once per round.
# The table gives each benchmark's best time over the rounds as a multiple of
# C Lua's best (below 1× is faster than C Lua).  A benchmark a revision fails
# to run is shown as "-" and reported, and the geomean is over the benchmarks
# every revision ran.

set -e
rounds=3
lua=${GOLUA_BENCH_LUA:-lua}
while getopts n:l: opt; do
	case $opt in
	n) rounds=$OPTARG ;;
	l) lua=$OPTARG ;;
	*) exit 2 ;;
	esac
done
shift $((OPTIND - 1))
[ $# -gt 0 ] || { echo "usage: $0 [-n rounds] [-l lua] REV..." >&2; exit 2; }
lua=$(command -v "$lua") || { echo "$0: no C Lua to compare with" >&2; exit 1; }

top=$(git rev-parse --show-toplevel)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

i=0
for rev in "$@"; do
	i=$((i + 1))
	echo "building $rev" >&2
	mkdir "$work/$i"
	git -C "$top" archive "$rev" | tar -x -C "$work/$i"
	rm -rf "$work/$i/benchmarks"
	cp -R "$top/benchmarks" "$work/$i/benchmarks"
	pgo=off
	[ -f "$work/$i/default.pgo" ] && pgo="$work/$i/default.pgo"
	(cd "$work/$i" && go test -c -pgo="$pgo" -tags noquotas -o "$work/$i.test" ./benchmarks)
done

for round in $(seq "$rounds"); do
	i=0
	for rev in "$@"; do
		i=$((i + 1))
		echo "round $round/$rounds: $rev" >&2
		if [ $i -eq 1 ]; then ref=$lua; else ref=; fi
		(cd "$top/benchmarks" && GOLUA_BENCH_LUA=$ref "$work/$i.test" \
			-test.run '^$' -test.bench . -test.count 1) >"$work/out" || true
		sed -n "s|^--- FAIL: BenchmarkStandard/\([^/]*\)/impl=golua.*|$rev fails \1|p" "$work/out" >>"$work/fails"
		sed -n "s|^BenchmarkStandard/\([^/]*\)/impl=\([a-z]*\)[^[:space:]]*[[:space:]].*[[:space:]]\([0-9.]*\) ns/op.*|\1 \2 $i \3|p" \
			"$work/out" >>"$work/times"
	done
done

[ -f "$work/fails" ] && sort -u "$work/fails" >&2

# Lines in times: benchmark, impl (golua or lua), revision number, ns/op.
awk -v revs="$*" '
	{
		key = $1 SUBSEP ($2 == "lua" ? 0 : $3)
		if (!(key in best) || $4 < best[key]) best[key] = $4
		if (!($1 in seen)) { seen[$1] = 1; order[++n] = $1 }
	}
	END {
		nrev = split(revs, rev, " ")
		printf "| benchmark |"
		for (r = 1; r <= nrev; r++) printf " %s |", rev[r]
		printf "\n|---|"
		for (r = 1; r <= nrev; r++) printf "---|"
		printf "\n"
		for (b = 1; b <= n; b++) {
			name = order[b]
			all = ((name, 0) in best)
			for (r = 1; r <= nrev; r++) if (!((name, r) in best)) all = 0
			if (all) common++
			printf "| %s |", name
			for (r = 1; r <= nrev; r++) {
				if (!((name, r) in best) || !((name, 0) in best)) {
					printf " - |"
					continue
				}
				x = best[name, r] / best[name, 0]
				printf " %.2f× |", x
				if (all) logsum[r] += log(x)
			}
			printf "\n"
		}
		printf "| **geomean of %d** |", common
		for (r = 1; r <= nrev; r++) printf " **%.2f×** |", exp(logsum[r] / common)
		printf "\n"
	}' "$work/times"
