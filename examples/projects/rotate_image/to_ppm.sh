#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 || $# -gt 2 ]]; then
  echo "usage: $0 INPUT_IMAGE [OUTPUT_PPM]" >&2
  exit 2
fi

input=$1
output=${2:-}

if [[ ! -f "$input" ]]; then
  echo "input image not found: $input" >&2
  exit 1
fi

if ! command -v convert >/dev/null 2>&1; then
  echo "ImageMagick 'convert' command is required" >&2
  exit 1
fi

if [[ -z "$output" ]]; then
  base=${input##*/}
  output=${base%.*}.ppm
fi

convert "$input" -compress none "$output"
echo "wrote $output"
