#!/bin/sh
set -eu

fail() {
	printf '%s\n' "$1" >&2
	exit 1
}

trimmed_instruction() {
	awk '
	function directive(line, value) {
		value = line
		sub(/^[ \t]+/, "", value)
		sub(/[ \t].*$/, "", value)
		return tolower(value)
	}
	{ print directive($0) }'
}

dockerfile=${1:-Dockerfile}
[ -f "$dockerfile" ] || fail "Dockerfile not found: $dockerfile"

# Keep only the last FROM stage, recognizing instruction keywords case-insensitively.
final_stage=$(awk '
	function directive(line, value) {
		value = line
		sub(/^[ \t]+/, "", value)
		sub(/[ \t].*$/, "", value)
		return tolower(value)
	}
	{
		if (directive($0) == "from") {
			stage = $0 ORS
		} else if (stage != "") {
			stage = stage $0 ORS
		}
	}
	END { printf "%s", stage }
' "$dockerfile")

[ -n "$final_stage" ] || fail 'final Dockerfile stage is missing'

final_instruction_value() {
	name=$1
	printf '%s\n' "$final_stage" | awk -v name="$name" '
	function directive(line, value) {
		value = line
		sub(/^[ \t]+/, "", value)
		sub(/[ \t].*$/, "", value)
		return tolower(value)
	}
	function value_after_directive(line, value) {
		value = line
		sub(/^[ \t]+/, "", value)
		sub(/^[^ \t]+[ \t]+/, "", value)
		return value
	}
	directive($0) == name { value = value_after_directive($0) }
	END { print value }
	'
}

has_final_directive_matching() {
	name=$1
	pattern=$2
	printf '%s\n' "$final_stage" | awk -v name="$name" -v pattern="$pattern" '
	function directive(line, value) {
		value = line
		sub(/^[ \t]+/, "", value)
		sub(/[ \t].*$/, "", value)
		return tolower(value)
	}
	directive($0) == name && $0 ~ pattern { found = 1 }
	END { exit !found }
	'
}

final_from=$(final_instruction_value from)
[ "$final_from" = 'alpine:3.22.6@sha256:5291449c3df73caf6ed85e649dec1b9e818b39a5d8c871e97afc13e9cd5e8fa8' ] \
	|| fail 'final stage must use the pinned Alpine 3.22.6 digest'

final_workdir=$(final_instruction_value workdir)
[ "$final_workdir" = '/app' ] || fail 'effective final WORKDIR must be /app'

final_user=$(final_instruction_value user)
user_component=${final_user%%:*}
case "$user_component" in
	'' | root | 0)
		fail 'effective final USER must be a dedicated non-root user'
		;;
esac

if ! has_final_directive_matching run 'apk[[:space:]].*add.*[[:space:]]curl([[:space:]]|\\|$)'; then
	fail 'final stage missing explicit curl healthcheck dependency'
fi
if ! has_final_directive_matching copy '--from=builder.*[[:space:]]/app/migrations[[:space:]]+[.]?/?migrations$'; then
	fail 'final stage missing migrations copy from the builder stage'
fi

healthcheck_count=$(printf '%s\n' "$final_stage" | awk '
	function directive(line, value) {
		value = line
		sub(/^[ \t]+/, "", value)
		sub(/[ \t].*$/, "", value)
		return tolower(value)
	}
	directive($0) == "healthcheck" { count++ }
	END { print count + 0 }
')
[ "$healthcheck_count" -eq 1 ] || fail 'effective final HEALTHCHECK must be present exactly once'
healthcheck=$(final_instruction_value healthcheck)
case "$healthcheck" in
	[Nn][Oo][Nn][Ee]) fail 'effective final HEALTHCHECK must not be NONE' ;;
esac
case "$healthcheck" in
	*'--interval=30s'*'--timeout=5s'*'--start-period=10s'*'--retries=3'*) ;;
	*) fail 'effective final HEALTHCHECK must use 30s interval, 5s timeout, 10s start period, and 3 retries' ;;
esac
case "$healthcheck" in
	*'CMD curl -fsS http://127.0.0.1:${PORT:-8080}/health || exit 1') ;;
	*) fail 'effective final HEALTHCHECK must probe /health using runtime PORT with an 8080 default' ;;
esac

final_cmd=$(final_instruction_value cmd)
[ "$final_cmd" = '["./app"]' ] || fail 'effective final CMD must be exec-form ["./app"]'

printf '%s\n' 'Dockerfile runtime contract checks passed'
