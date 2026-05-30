# Naming

Trux source-level identifiers use lower camelCase.

This applies to public language surface names, including compiler-recognized builtins:

```trux
readLine()
readInt()
readFloat()
readBool()
readFile(path)
writeFile(path, contents)
readCsv(path, columns)
writeCsv(path, cells, columns)
```

Acronyms are treated as normal words in source identifiers, so use `readCsv` and `writeCsv`, not `readCSV` or `writeCSV`.

Host implementation names follow the conventions of the host language. Internal Go names may use Go-style initialisms such as `IOCallReadCSV`, and runtime C helpers may use snake_case names such as `rt_read_line`. Those names are implementation details and are not part of the Trux source language surface.

## Tradeoff

Lower camelCase keeps examples, builtin names, and future package member names visually consistent. The cost is a breaking rename from the earlier snake_case IO builtins. Keeping snake_case aliases would avoid that break, but it would leave two valid spellings for the same operation and weaken the language convention.
