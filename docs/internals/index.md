# How it works

This section is for people who want to understand, debug, or change the compiler. You don't need any of it to *use*
Trux.

- **[Compiling to C and CUDA](compilation.md)**: what happens to a `.tx` file, stage by stage, and what the generated C
  and CUDA look like.
- **[Codebase tour](architecture.md)**: how the Go repository is organized and where each stage lives.
- **[Contributing](contributing.md)**: building, testing, and the conventions to follow when you change the compiler or
  these docs.

The [design notes](../design/index.md) collect the longer write-ups behind individual decisions (arenas, modules, the GPU
model, naming, and more).

## The big picture

```mermaid
flowchart TD
    subgraph Front end
        L["modules.Load<br/>(internal/modules)"] -->|"for each file"| P["lexer + parser<br/>(internal/lexer, internal/parser)"]
        P --> M["validate imports and visibility,<br/>rename private functions,<br/>merge into one ast.Program"]
    end
    subgraph Middle
        M --> T["types.Check<br/>(internal/types)"]
        T --> I["ir.Build<br/>(internal/ir)"]
    end
    subgraph Back end
        I --> G["codegen C<br/>(internal/codegen/c)"]
        R["runtime headers<br/>(internal/runtime/c)"] --> W
        G --> W["write .c or .cu<br/>plus headers"]
        W --> CC["cc / $CC<br/>or nvcc / $NVCC"]
    end
    CC --> X["native executable"]
```

The `trux` command (`cmd/trux`) drives these stages. All of the compiler is ordinary Go code with one external
dependency, the Cobra CLI library.
