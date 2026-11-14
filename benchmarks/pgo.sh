#!/bin/sh
# Make default.pgo, the CPU profile that the Go compiler optimises golua with
# (profile-guided optimisation, PGO).
#
#   benchmarks/pgo.sh [-t benchtime]
#
# A build of the golua command picks up the default.pgo beside main.go by
# itself; a program that embeds golua has to be given it (see the README).
# The compiler uses the profile to inline the calls that are hot and to turn
# hot calls through an interface into direct ones, so the profile should be
# of golua doing typical work: here, running each benchmark for the same
# time (3s by default), in the build with quotas and in the one without,
# both compiled without a profile.
#
# The profile names functions and the lines of their calls, so it goes stale
# as the code changes: the calls it no longer finds are simply not optimised.
# Make it again after substantial changes to the run loop, and before a
# release.

set -e
benchtime=3s
while getopts t: opt; do
	case $opt in
	t) benchtime=$OPTARG ;;
	*) echo "usage: $0 [-t benchtime]" >&2; exit 2 ;;
	esac
done

root=$(cd "$(dirname "$0")/.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

cd "$root"
for build in quotas noquotas; do
	tags=
	[ $build = noquotas ] && tags=noquotas
	echo "profiling the $build build..." >&2
	go test -c -pgo=off -tags "$tags" -o "$work/$build.test" ./benchmarks
	(cd benchmarks && "$work/$build.test" -test.run '^$' -test.bench '//^impl=golua$' \
		-test.benchtime "$benchtime" -test.cpuprofile "$work/$build.pprof" >/dev/null)
done
go tool pprof -proto "$work/quotas.pprof" "$work/noquotas.pprof" >default.pgo
echo "wrote $root/default.pgo" >&2
