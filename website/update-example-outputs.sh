#!/usr/bin/env bash
# Regenerates website/outputs/*.txt by running the deterministic CPU examples
# with the compiler built from this checkout. The documentation site includes
# these files, so the outputs it shows are always real program output.
#
# Usage (from anywhere):  website/update-example-outputs.sh
# Check mode for CI:      website/update-example-outputs.sh --check
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

mode="write"
if [[ "${1:-}" == "--check" ]]; then
  mode="check"
fi

bin="$(mktemp -d)/trux"
go build -o "$bin" ./cmd/trux

out_dir="website/outputs"
tmp_dir="$(mktemp -d)"
mkdir -p "$out_dir"

# name|stdin|command arguments
examples=(
  "hello||run examples/hello.tx"
  "arithmetic||run examples/arithmetic.tx"
  "primitives||run examples/primitives.tx"
  "strings||run examples/strings.tx"
  "control_flow||run examples/control_flow.tx"
  "else_if||run examples/else_if.tx"
  "loops||run examples/loops.tx"
  "return_paths||run examples/return_paths.tx"
  "fibonacci||run examples/fibonacci.tx"
  "assertions||run examples/assertions.tx"
  "collections||run examples/collections.tx"
  "jagged_lists||run examples/jagged_lists.tx"
  "nested_slices||run examples/nested_slices.tx"
  "matrix_multiply_2x2||run examples/matrix_multiply_2x2.tx"
  "ownership_clone||run examples/ownership_clone.tx"
  "mut_parameters||run examples/mut_parameters.tx"
  "modules_basic||run examples/modules/basic/main.tx"
  "modules_transitive||run examples/modules/transitive/main.tx"
  "modules_private_names||run examples/modules/private_names/main.tx"
  "modules_same_package_math||run examples/modules/same_package_math/main.tx"
  "modules_mut_parameters||run examples/modules/mut_parameters/main.tx"
  "scorebook||run examples/projects/scorebook/main.tx"
  "io_basics|Ada\n41\n|run examples/projects/io_basics/main.tx"
  "interactive_counter|Ada\n7\n|run examples/projects/interactive_counter/main.tx"
  "file_notes|review IO examples\n|run examples/projects/file_notes/main.tx"
  "csv_roster|Cara\nops\n|run examples/projects/csv_roster/main.tx"
  "csv_quoting||run examples/projects/csv_quoting/main.tx"
)

for entry in "${examples[@]}"; do
  IFS='|' read -r name input args <<<"$entry"
  # shellcheck disable=SC2086
  printf "$input" | "$bin" $args >"$tmp_dir/$name.txt"
done

# Generated C/CUDA excerpts shown on the "Compiling to C and CUDA" page.
mkdir -p "$tmp_dir/generated"
emit_dir="$(mktemp -d)"
"$bin" emit-c --out-dir "$emit_dir/hello" examples/hello.tx >/dev/null
cp "$emit_dir/hello/hello.c" "$tmp_dir/generated/hello.c"
"$bin" emit-c --out-dir "$emit_dir/matmul" examples/gpu_matrix_multiply.tx >/dev/null
cu="$emit_dir/matmul/gpu_matrix_multiply.cu"
# The kernel definition: from its "#line" marker to the closing brace.
awk '/^__global__ .*\{$/ {print prev; on=1} on {print} on && /^}$/ {exit} {prev=$0}' "$cu" \
  >"$tmp_dir/generated/matmul_kernel.cu"
# The host-side launch and its error check.
grep -A1 '<<<' "$cu" | sed 's/^ *//' >"$tmp_dir/generated/matmul_launch.cu"
rm -rf "$emit_dir"

# The example projects write these files; keep the checkout clean.
rm -f examples/projects/io_basics/copy.txt examples/projects/io_basics/output.csv \
  examples/projects/file_notes/summary.txt examples/projects/csv_roster/out.csv \
  examples/projects/csv_quoting/out.csv

if [[ "$mode" == "check" ]]; then
  if ! diff -r "$out_dir" "$tmp_dir"; then
    echo "website/outputs is stale; run website/update-example-outputs.sh" >&2
    exit 1
  fi
  echo "website/outputs is up to date"
else
  rm -rf "$out_dir"
  cp -r "$tmp_dir" "$out_dir"
  echo "wrote $(find "$out_dir" -type f | wc -l) files to $out_dir"
fi
