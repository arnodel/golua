#!/bin/sh
# Make default.pgo, the CPU profile that the Go compiler optimises golua with
# (profile-guided optimisation, PGO).
#
#   benchmarks/pgo.sh [-t benchtime] [-l limit]   make the profile
#   benchmarks/pgo.sh -c                          check that it is up to date
#
# A build of the golua command picks up the default.pgo beside main.go by
# itself; a program that embeds golua has to be given it (see the README).
# The compiler uses the profile to inline the calls that are hot and to turn
# hot calls through an interface into direct ones, so the profile should be
# of golua doing typical work: here, running each benchmark for the same
# time (3s by default), in the build with quotas and in the one without,
# both compiled without a profile.  A run of the benchmarks normally takes a
# minute and a half; one that takes longer than the limit, in seconds (600
# by default), is stopped, and no profile is written: a run that crawls, for
# whatever reason, would make a profile of mostly one benchmark.
#
# The profile names functions and the lines of their calls, counted from the
# start of the function, so it goes stale as the code changes: the calls it
# no longer finds are simply not optimised.  The function that matters most
# is the run loop, and what the compiler does inside it is lost when its
# lines move.  The profile is 90K that git cannot store as a difference, and
# the run loop changes often, so the profile is not made again with every
# such change but for each release: pgo.sum records the run loop the profile
# was made for, and -c, which CI runs on a release tag, fails if the run loop
# has changed since.

set -e
benchtime=3s
limit=600
check=
while getopts cl:t: opt; do
	case $opt in
	c) check=1 ;;
	l) limit=$OPTARG ;;
	t) benchtime=$OPTARG ;;
	*) echo "usage: $0 [-t benchtime] [-l limit] | -c" >&2; exit 2 ;;
	esac
done

root=$(cd "$(dirname "$0")/.." && pwd)

# fingerprint prints a hash of the source of the run loop.
fingerprint() {
	awk '/^func \(c \*LuaCont\) RunInThread\(/,/^}/' "$root/runtime/luacont.go" |
		git hash-object --stdin
}

# limited runs a command and stops it if it takes more than $limit seconds.
# The benchmarks cannot be given the limit themselves: -test.timeout does not
# cover benchmarks.
limited() {
	"$@" &
	cmd=$!
	(sleep "$limit" && kill "$cmd") 2>/dev/null &
	watchdog=$!
	status=0
	wait "$cmd" || status=$?
	pkill -P "$watchdog" 2>/dev/null || true
	kill "$watchdog" 2>/dev/null || true
	return $status
}

if [ -n "$check" ]; then
	if [ "$(fingerprint)" = "$(cat "$root/benchmarks/pgo.sum" 2>/dev/null)" ]; then
		echo "default.pgo is up to date with the run loop" >&2
		exit 0
	fi
	echo "default.pgo is stale: the run loop has changed since it was made." >&2
	echo "Make it again with benchmarks/pgo.sh" >&2
	exit 1
fi

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

cd "$root"
for build in quotas noquotas; do
	tags=
	[ $build = noquotas ] && tags=noquotas
	echo "profiling the $build build..." >&2
	go test -c -pgo=off -tags "$tags" -o "$work/$build.test" ./benchmarks
	if ! (cd benchmarks && limited "$work/$build.test" -test.run '^$' \
		-test.bench '//^impl=golua$' -test.benchtime "$benchtime" \
		-test.cpuprofile "$work/$build.pprof" >/dev/null); then
		echo "$0: the benchmarks failed or took more than ${limit}s: no profile written" >&2
		exit 1
	fi
done
go tool pprof -proto "$work/quotas.pprof" "$work/noquotas.pprof" >default.pgo
fingerprint >benchmarks/pgo.sum
echo "wrote $root/default.pgo" >&2
