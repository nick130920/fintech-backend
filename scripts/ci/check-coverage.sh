#!/usr/bin/env bash
set -euo pipefail

coverage_file=${1:-coverage.out}
baseline_file=${2:-.ci/coverage-baseline.txt}

if [[ ! -f "$coverage_file" ]]; then
	printf 'coverage profile not found: %s\n' "$coverage_file" >&2
	exit 1
fi
if [[ ! -f "$baseline_file" ]]; then
	printf 'coverage baseline not found: %s\n' "$baseline_file" >&2
	exit 1
fi

baseline=$(tr -d '[:space:]' <"$baseline_file")
if [[ ! "$baseline" =~ ^[0-9]+([.][0-9]+)?$ ]]; then
	printf 'coverage baseline must be a numeric percentage: %s\n' "$baseline" >&2
	exit 1
fi

summary=$(go tool cover -func="$coverage_file" | awk '/^total:/ { print $3; found = 1 } END { if (!found) exit 1 }')
if [[ ! "$summary" =~ ^[0-9]+([.][0-9]+)?%$ ]]; then
	printf 'unable to parse total coverage from %s: %s\n' "$coverage_file" "$summary" >&2
	exit 1
fi

coverage=${summary%%%}
if ! awk -v coverage="$coverage" -v baseline="$baseline" 'BEGIN { exit !(coverage >= baseline) }'; then
	printf 'coverage %s%% is below committed baseline %s%%\n' "$coverage" "$baseline" >&2
	exit 1
fi

printf 'coverage %s%% meets committed baseline %s%%\n' "$coverage" "$baseline"
