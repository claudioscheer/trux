# Example Projects

These examples are small IO-focused programs with their own input fixtures.

Run them from the repository root after building `trux`.

```bash
printf 'Ada\n41\n' | trux run examples/projects/io_basics/main.tx
printf 'Ada\n7\n' | trux run examples/projects/interactive_counter/main.tx
printf 'review IO examples\n' | trux run examples/projects/file_notes/main.tx
printf 'Cara\nops\n' | trux run examples/projects/csv_roster/main.tx
trux run examples/projects/rotate_image/main.tx
```

`rotate_image` reads a PPM image, rotates it on CPU and GPU, writes both
outputs, and compares the results.
