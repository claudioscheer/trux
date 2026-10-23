#!/usr/bin/env bash
# Checks the Markdown versions of the docs pages after a Hugo build:
#   1. no shortcode syntax ({{< or {{%) is left in any generated .md or
#      llms*.txt file (outside inline code), and
#   2. every link in llms.txt points at a file that exists in the build.
#
# Usage: website/check-markdown-output.sh [public-dir]   (default: website/public)
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
public="${1:-$root/public}"
base_url="https://thea1lab.github.io/trux/"
status=0

mapfile -t files < <(find "$public" -name '*.md' -o -name 'llms*.txt' | sort)
if [[ ${#files[@]} -eq 0 ]]; then
  echo "no Markdown output found in $public" >&2
  exit 1
fi

for f in "${files[@]}"; do
  # Drop inline code spans, where shortcode syntax is shown on purpose.
  if leftovers="$(sed -E 's/`[^`]*`//g' "$f" | grep -n -E '\{\{[<%]')"; then
    echo "unconverted shortcode syntax in ${f#"$public"/}:" >&2
    echo "$leftovers" >&2
    status=1
  fi
done

links=0
while read -r link; do
  url="${link#"]("}"
  url="${url%")"}"
  path="${url#"$base_url"}"
  links=$((links + 1))
  if [[ ! -f "$public/$path" ]]; then
    echo "llms.txt links to a missing file: $url" >&2
    status=1
  fi
done < <(grep -o -E "\]\(${base_url}[^)]+\)" "$public/llms.txt")

if [[ $status -eq 0 ]]; then
  echo "checked ${#files[@]} Markdown files and $links llms.txt links"
fi
exit $status
