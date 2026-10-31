#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

args=()
out_seen=0
while (($#)); do
	if [[ "$1" == -out ]]; then
		if (($# < 2)); then
			echo '-out の出力先がありません' >&2
			exit 2
		fi
		out_seen=1
		out_path="$2"
		mkdir -p "$(dirname "$out_path")"
		out_path="$(realpath -m "$out_path")"
		args+=("-out" "$(wslpath -w "$out_path")")
		shift 2
	else
		args+=("$1")
		shift
	fi
done
if ((out_seen == 0)); then
	echo '-out で PNG の出力先を指定してください' >&2
	exit 2
fi

cd "$repo_dir"
GOOS=windows go build -buildvcs=false -o "$tmp_dir/shot.exe" ./tools/shot
"$tmp_dir/shot.exe" "${args[@]}"
