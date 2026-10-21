# Programs that fail on purpose

Each program here shows one common mistake. They are **expected to fail**: some are rejected by the compiler, some stop
at run time with a `trux runtime error`, and two (`gpu_coordinate_on_host.tx`, `gpu_buffers_without_kernel.tx`) show
known compiler bugs where the C compiler rejects the generated code.

`website/update-example-outputs.sh` runs every program listed in its `errors` array, checks that it exits with a
non-zero status, and records the output in `website/outputs/errors/`. The language guide shows those recordings, and CI
runs the script with `--check`, so the documented messages always match the compiler.

To add a case: write the program, add a line to the `errors` array, run `website/update-example-outputs.sh`, and include
both files from the docs page.

The `.tx` entry files of multi-file cases are named `program.tx` instead of `main.tx`, so the Go tests that compile every
`examples/**/main.tx` skip them.
