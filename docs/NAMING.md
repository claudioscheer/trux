# Naming

Trux source-level identifiers use lower camelCase.

This applies to public language surface names, including standard package members:

```trux
io.readLine()
io.readInt()
io.readFloat()
io.readBool()
io.readFile(path)
io.writeFile(path, contents)
csv.read(path, columns)
csv.write(path, cells, columns)
```

Package names stay short and lowercase. The old global `readCsv`/`writeCsv` names are replaced by `csv.read`/`csv.write`, so there is no source-level acronym in the CSV API.

Host implementation names follow the conventions of the host language. Internal Go names may use Go-style initialisms such as `IOCallReadCSV`, and runtime C helpers may use snake_case names such as `rt_read_line`. Those names are implementation details and are not part of the Trux source language surface.

## Tradeoff

Lower camelCase keeps examples, builtin names, and package member names visually consistent. Moving IO behind `io` and `csv` is a breaking change from the earlier global IO names. Keeping aliases would avoid that break, but it would leave two valid spellings for the same operation and weaken the package boundary.
