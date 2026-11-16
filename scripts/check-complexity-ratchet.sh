#!/usr/bin/env bash
# R-136: the cyclomatic complexity of a function may not grow past its record in
# metrics/cyclomatic.tsv. New functions are added to the record by --update.
# Input: lizard --csv output on stdin.
set -euo pipefail

record="$(dirname "$0")/../metrics/cyclomatic.tsv"
update=false
[[ "${1:-}" == "--update" ]] && update=true

current="$(mktemp)"
trap 'rm -f "$current"' EXIT
# lizard csv: nloc,ccn,token,param,length,location,file,function,long_name,start,end
awk -F, '{ gsub(/"/, ""); printf "%s\t%s\t%s\n", $7, $8, $2 }' | sort > "$current"

if $update; then
  { printf 'path\tfunction\tvalue\n'; cat "$current"; } > "$record"
  echo "✓ complexity record updated: $(wc -l < "$current" | tr -d ' ') functions"
  exit 0
fi

failed=0
while IFS=$'\t' read -r path function value; do
  known="$(awk -F'\t' -v p="$path" -v f="$function" '$1 == p && $2 == f { print $3 }' "$record")"
  if [[ -n "$known" && "$value" -gt "$known" ]]; then
    echo "✗ complexity grew: $path $function $known → $value"
    failed=1
  fi
done < "$current"

if [[ "$failed" -ne 0 ]]; then
  exit 1
fi
echo "✓ complexity ratchet: no function grew"
