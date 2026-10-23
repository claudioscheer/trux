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
# The launch, gpu.sync, and copy back, as the GPU guide explains them.
awk '/<<</ {on=1} on {print} on && /rt_gpu_copy_to_host/ {exit}' "$cu" | sed 's/^ *//' \
  >"$tmp_dir/generated/matmul_sync.cu"
# The runtime helpers behind gpu.sync and gpu.to (int version).
rt_header="$emit_dir/matmul/trux_runtime.h"
for helper in rt_gpu_copy_to_host_int rt_gpu_sync; do
  awk -v name="$helper" '$0 ~ "void " name "\\(" {on=1} on {print} on && /^}$/ {exit}' "$rt_header"
done >"$tmp_dir/generated/gpu_sync_runtime.h"
rm -rf "$emit_dir"

# Failing programs under examples/errors. Each must exit non-zero; the
# recorded text is exactly what the compiler or program printed, so the
# error messages quoted in the docs cannot drift from the real ones.
# Paths are made relative to the checkout, the random `trux run` temp dir is
# replaced with <id>, and terminal colors are removed.
# name|stdin|command arguments|filter
# The "cc" filter keeps only the C compiler's hard errors. Implicit function
# declarations are dropped because older GCC releases report them as warnings.
errors=(
  "program_missing_main||run examples/errors/program_missing_main.tx|"
  "program_main_params||run examples/errors/program_main_params.tx|"
  "program_import_order||run examples/errors/program_import_order.tx|"
  "types_mixed_numbers||run examples/errors/types_mixed_numbers.tx|"
  "types_int_to_float||run examples/errors/types_int_to_float.tx|"
  "types_if_int||run examples/errors/types_if_int.tx|"
  "types_bad_escape||run examples/errors/types_bad_escape.tx|"
  "variables_duplicate||run examples/errors/variables_duplicate.tx|"
  "variables_undefined||run examples/errors/variables_undefined.tx|"
  "variables_compound||run examples/errors/variables_compound.tx|"
  "operators_overflow||run examples/errors/operators_overflow.tx|"
  "operators_divide_by_zero||run examples/errors/operators_divide_by_zero.tx|"
  "operators_print_value||run examples/errors/operators_print_value.tx|"
  "operators_compare_collections||run examples/errors/operators_compare_collections.tx|"
  "functions_missing_return||run examples/errors/functions_missing_return.tx|"
  "functions_no_return_type||run examples/errors/functions_no_return_type.tx|"
  "functions_argument_type||run examples/errors/functions_argument_type.tx|"
  "functions_mutate_parameter||run examples/errors/functions_mutate_parameter.tx|"
  "functions_mut_scalar||run examples/errors/functions_mut_scalar.tx|"
  "functions_mut_borrowed||run examples/errors/functions_mut_borrowed.tx|"
  "control_if_same_line||run examples/errors/control_if_same_line.tx|"
  "control_break_outside||run examples/errors/control_break_outside.tx|"
  "control_assert||run examples/errors/control_assert.tx|"
  "collections_index_out_of_bounds||run examples/errors/collections_index_out_of_bounds.tx|"
  "collections_slice_out_of_bounds||run examples/errors/collections_slice_out_of_bounds.tx|"
  "collections_array_length||run examples/errors/collections_array_length.tx|"
  "collections_append_array||run examples/errors/collections_append_array.tx|"
  "collections_string_assign||run examples/errors/collections_string_assign.tx|"
  "memory_unknown_ownership||run examples/errors/memory_unknown_ownership.tx|"
  "modules_bare_import||run examples/errors/modules_bare_import.tx|"
  "modules_private||run examples/errors/modules_private/program.tx|"
  "modules_transitive||run examples/errors/modules_transitive/program.tx|"
  "modules_cycle||run examples/errors/modules_cycle/program.tx|"
  "testing_rules||test examples/errors/testing_rules/math.test.tx|"
  "testing_failure||test examples/errors/testing_failure/math.test.tx|"
  "gpu_kernel_return||run examples/errors/gpu_kernel_return.tx|"
  "gpu_kernel_return_value||run examples/errors/gpu_kernel_return_value.tx|"
  "gpu_kernel_string||run examples/errors/gpu_kernel_string.tx|"
  "gpu_kernel_call||run examples/errors/gpu_kernel_call.tx|"
  "gpu_kernel_print||run examples/errors/gpu_kernel_print.tx|"
  "gpu_sync_in_kernel||run examples/errors/gpu_sync_in_kernel.tx|"
  "gpu_alloc_untyped||run examples/errors/gpu_alloc_untyped.tx|"
  "gpu_buffer_host_index||run examples/errors/gpu_buffer_host_index.tx|"
  "gpu_element_type||run examples/errors/gpu_element_type.tx|"
  "gpu_launch1d_coordinate||run examples/errors/gpu_launch1d_coordinate.tx|"
  "gpu_launch_arguments||run examples/errors/gpu_launch_arguments.tx|"
  "gpu_launch_two_shapes||run examples/errors/gpu_launch_two_shapes.tx|"
  "gpu_coordinate_on_host||run examples/errors/gpu_coordinate_on_host.tx|cc"
  "gpu_buffers_without_kernel||run examples/errors/gpu_buffers_without_kernel.tx|cc"
  "io_read_int|abc\n|run examples/errors/io_read_int.tx|"
  "io_missing_file||run examples/errors/io_missing_file.tx|"
  "io_not_imported||run examples/errors/io_not_imported.tx|"
  "csv_columns||run examples/errors/csv_columns.tx|"
  "time_negative_sleep||run examples/errors/time_negative_sleep.tx|"
  "image_pixel_count||run examples/errors/image_pixel_count.tx|"
)

mkdir -p "$tmp_dir/errors"
for entry in "${errors[@]}"; do
  IFS='|' read -r name input args filter <<<"$entry"
  status=0
  # shellcheck disable=SC2086
  printf "$input" | LC_ALL=C "$bin" $args >"$tmp_dir/errors/$name.raw" 2>&1 || status=$?
  if [[ "$status" == 0 ]]; then
    echo "examples/errors case $name was expected to fail but exited 0" >&2
    exit 1
  fi
  sed -e "s#$root/##g" \
    -e 's#/[^ ]*/trux-run-[0-9]*/#/tmp/trux-run-<id>/#g' \
    -e $'s/\x1b\\[[0-9;]*m//g' \
    "$tmp_dir/errors/$name.raw" >"$tmp_dir/errors/$name.norm"
  if [[ "$filter" == "cc" ]]; then
    grep -E '^cc failed|error:' "$tmp_dir/errors/$name.norm" |
      grep -v 'implicit declaration' >"$tmp_dir/errors/$name.txt"
  else
    cp "$tmp_dir/errors/$name.norm" "$tmp_dir/errors/$name.txt"
  fi
  rm -f "$tmp_dir/errors/$name.raw" "$tmp_dir/errors/$name.norm"
done

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
