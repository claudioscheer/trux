# Tiny Transformer on Tiny Shakespeare

**Date:** 2026-10-23
**Status:** Approved for implementation

## Summary

Train a very small character-level decoder-only transformer on Tiny Shakespeare
with Python and PyTorch, export the complete model and validation fixtures into
one TXM file, and run the same weights in Trux on the CPU and the GPU. Compare
the final logits and generated tokens with a Python reference and with each
other, then report timing for both paths.

## Goals

- Show a real transformer running end to end in Trux, not a toy
  attention-shaped approximation.
- Keep the model small enough to load the whole thing once and hold it in host
  and device memory for the entire run.
- Make correctness checkable value by value, not just by looking at generated
  text.
- Run the same prompt and decoding on the CPU and the GPU and compare both
  output and performance.
- Add the missing host math primitives to Trux as a proper `math` module.

## Non-goals

- Training inside Trux. Training and export are Python only.
- Downloading a pretrained checkpoint. The model is trained from scratch.
- KV caching, batched inference, quantization, or a float32 GPU path.
- Temperature or top-k sampling in the first version. Greedy decoding is used
  so Python, CPU, and GPU can be compared exactly.
- A CPU-only Trux build. `gpu.mount` still requires the CUDA build, matching the
  existing MNIST benchmark. The CPU path uses the host copy of the mounted model
  and does not launch kernels.

## Success criteria

1. `train.py` trains a character-level model on Tiny Shakespeare and writes one
   `model.txm` smaller than 5 MB containing every weight and every fixture.
2. The Python reference is computed in float64 from the exact values written to
   the TXM, not from a separate float32 checkpoint.
3. The CPU and GPU paths each match every Python prompt logit and every Python
   step logit within `1e-9`, and match every greedy token exactly.
4. The CPU and GPU generated strings are identical.
5. The program prints the number of values checked, the maximum absolute
   difference for each path, both generated strings, and prefill/decode timing.
6. `make transformer` runs without Python or PyTorch installed.
7. The new `math` module has compiler tests and documentation.

## Why this design

The first working path is the CPU forward pass because it is easy to inspect and
debug one scalar loop at a time. The GPU path reuses the same TXM and the same
validation fixtures, so a mismatch points at the GPU implementation rather than
at a different model. Float64 everywhere, including GPU buffers, keeps the
Python/Trux comparison tight. The cost is lower FP64 throughput on the GTX 1660;
that is part of what the benchmark reports.

The alternatives considered were GPU-first (much harder to debug the first
correct result), a pretrained checkpoint (more capable but brings a tokenizer
and architecture export problem that is larger than the model itself), and
sampling from the start (requires a deterministic RNG shared by Python and
Trux). None of them serve the first goal better than the chosen path.

## Corpus and tokenizer

- Corpus: Tiny Shakespeare from
  `https://raw.githubusercontent.com/karpathy/char-rnn/master/data/tinyshakespeare/input.txt`.
  The training script downloads it into `examples/projects/tiny_transformer/data/`,
  which is gitignored. `--data` accepts a local file.
- Tokenizer: one token per character. The vocabulary is the sorted set of
  characters in the corpus. Tiny Shakespeare has 65 unique characters; the
  exporter writes the actual vocabulary size into `config`, and the Trux program
  checks the embedded vocabulary string against it.
- Split: first 90% for training, last 10% for validation.
- The vocabulary string is embedded in `main.tx`. The prompt token IDs and the
  expected logits in the TXM make a vocabulary mismatch fail validation
  immediately.

## Model

Decoder-only pre-layer-norm transformer:

```text
x = token_embedding[token] + position_embedding[position]
for each layer:
    a = layer_norm_1(x)
    q, k, v = a @ Wq^T + bq, a @ Wk^T + bk, a @ Wv^T + bv
    split q, k, v into heads
    scores = q @ k^T / sqrt(head_dim), causal mask
    attention = softmax(scores) @ v
    x = x + attention @ Wo^T + bo
    m = layer_norm_2(x)
    h = relu(m @ W1^T + b1)
    x = x + h @ W2^T + b2
x = final_layer_norm(x)
logits = x @ Wout^T + bout
```

Configuration:

| Name | Value |
| --- | --- |
| Layers | 2 |
| Heads | 4 |
| Model width | 128 |
| MLP width | 512 |
| Context length | 128 |
| Vocabulary | corpus-defined, expected 65 |
| Dropout | 0.0 |
| Weight tying | no |
| Parameters | about 430k |
| TXM size | about 3.5 MB in float64 |

The MLP uses ReLU, which keeps the model a standard transformer while avoiding
GELU's `erf`/`tanh` dependency. Position embeddings are learned absolute
embeddings. Attention is causal; query position `t` may only see keys `0..t`.

## Prompt and generation

- Prompt: `JULIET:\n` (8 characters).
- Generated tokens: 108.
- Total sequence: 116 tokens, inside the 128-token context, so the first
  version does not need a sliding window.
- Decoding: greedy argmax at every step.
- The generated string is decoded through the embedded vocabulary.

A tiny character model trained on Shakespeare produces Shakespeare-flavored
text, not guaranteed grammatical modern sentences. If greedy decoding falls
into a loop, that is a model-quality limitation, not a correctness failure. A
deterministic sampling mode can be added later without changing the validation
architecture.

## TXM artifact

`model.txm` holds every tensor needed by both paths. Layer weights are stacked
on a leading layer axis so the CPU can loop over layers and the GPU kernels can
take a layer offset.

| Tensor | Shape | Meaning |
| --- | --- | --- |
| `config` | `[6]` int | vocabulary, context, layers, heads, width, MLP width |
| `inv_width` | `[1]` float | `1 / D`, for layer norm |
| `attn_scale` | `[1]` float | `1 / sqrt(D / H)` |
| `ln_eps` | `[1]` float | layer norm epsilon |
| `tok_emb` | `[V, D]` | token embeddings |
| `pos_emb` | `[T, D]` | position embeddings |
| `ln1_g`, `ln1_b` | `[L, D]` | per-layer attention layer norm |
| `wq`, `bq` | `[L, D, D]`, `[L, D]` | query projection |
| `wk`, `bk` | `[L, D, D]`, `[L, D]` | key projection |
| `wv`, `bv` | `[L, D, D]`, `[L, D]` | value projection |
| `wo`, `bo` | `[L, D, D]`, `[L, D]` | attention output projection |
| `ln2_g`, `ln2_b` | `[L, D]` | per-layer MLP layer norm |
| `w1`, `b1` | `[L, M, D]`, `[L, M]` | MLP up projection |
| `w2`, `b2` | `[L, D, M]`, `[L, D]` | MLP down projection |
| `lnf_g`, `lnf_b` | `[D]` | final layer norm |
| `wout`, `bout` | `[V, D]`, `[V]` | output projection |
| `prompt_tokens` | `[P]` int | tokenized fixed prompt |
| `expected_prompt_logits` | `[P, V]` | Python logits for every prompt position |
| `expected_step_logits` | `[N, V]` | Python logits for every generation step |
| `expected_tokens` | `[N]` int | Python greedy token IDs |

`gpu.mount` parses the file once, keeps the host copy, and uploads all tensors to
the device. `gpu.hostFloats` and `gpu.floats` are views of that single loaded
model; there is no second copy on disk and no streaming.

The model file is committed so the example runs without PyTorch. The training
script and a gitignored `checkpoint.pt` allow regeneration.

## Python training and export

`examples/projects/tiny_transformer/train.py`:

- Uses PyTorch, NumPy, and the standard library only.
- Chooses CUDA when available, otherwise CPU.
- AdamW with learning rate `3e-4`, beta2 `0.95`, weight decay `0.1`, gradient
  clipping `1.0`, 100 warmup steps and cosine decay.
- Batch size 64, block size 128, 8,000 steps by default.
- Evaluates on a fixed validation sample every 250 steps and keeps the best
  validation loss in `checkpoint.pt`.
- Prints sample generations during training so model quality can be judged
  before export.
- After training, upcasts the best model to float64, evaluates it, and uses
  those float64 weights for both the TXM and the reference logits.
- Tokenizes the fixed prompt, runs the prompt forward, then generates 108 tokens
  greedily. At every step it records the final logits and the chosen token.
- Checks the top-2 logit margin at every generation step. If the margin is at or
  below `1e-7`, export fails with a clear message; a near-tie would make the
  token comparison meaningless.
- Writes `config`, all weights, prompt tokens, expected logits, and expected
  tokens with `python/txm.py`.
- Verifies the written file with `txm.verify` before exiting.

The exporter prints the generated reference text and the smallest top-2 margin
so the committed fixture can be reviewed.

## Trux math module

The host currently has no `exp` or `sqrt`; GPU kernels only have `exp`. Add a
small `math` standard-library package:

- `math.exp(x float) float`
- `math.sqrt(x float) float`

Host codegen emits the C library functions and the build links `libm`. Wrong
argument counts and non-float arguments are compile errors. `sqrt` of a negative
value follows IEEE/C behavior and produces NaN; layer-norm callers always pass
`variance + epsilon`.

Add a GPU kernel builtin `sqrt(x)` next to the existing `exp(x)`, with the same
float-argument checks. Kernels use `exp` and `sqrt` builtins; host code uses
`math.exp` and `math.sqrt`.

Add compiler/codegen tests for the new calls and documentation under
`website/content/docs/stdlib/math.md`, linked from the stdlib index.

## CPU inference

`examples/projects/tiny_transformer/main.tx` loads the model with `gpu.mount`,
reads the configuration, and obtains host views with `gpu.hostFloats` and
`gpu.hostInts`.

The CPU forward pass uses flat row-major arrays and scalar loops:

1. Copy token and position embeddings into the activation buffer.
2. For each layer:
   - layer norm into a scratch buffer;
   - Q, K, V projections;
   - attention scores and causal softmax;
   - weighted sum and output projection with residual;
   - second layer norm, MLP up + ReLU, MLP down with residual.
3. Final layer norm and vocabulary logits for the last position.

Scratch buffers are allocated once and reused. Tokenization walks the prompt
string and finds each character in the embedded vocabulary; the resulting IDs
must equal `prompt_tokens`.

Greedy generation keeps the growing token list and re-runs the full forward pass
for each new token. The sequence stays within the 128-token context, so no
sliding window or KV cache is needed.

## GPU inference

The GPU path uses the device views returned by `gpu.floats` and reuses the same
mounted weights. Kernels are deliberately simple: one thread per output element,
dot products in scalar loops, and a layer offset into the stacked weights.

Kernels:

1. embedding copy
2. layer norm
3. Q/K/V projections
4. attention scores
5. causal softmax
6. weighted value sum
7. attention output projection + residual
8. MLP up + ReLU
9. MLP down + residual
10. final layer norm + logits

The activation buffers are allocated once. Each generation step launches the
full set of kernels, synchronizes, copies the last logits row to a host slice,
and takes the argmax on the host. Timing includes the launches, synchronization,
and logit copy so the reported number is the cost of producing a token.

The GPU uses double-precision buffers, matching the TXM and the Python
reference. The GTX 1660 has weak FP64 throughput compared with FP32; the
benchmark reports the real result rather than hiding it behind a different
precision.

## Validation

For both paths, the program checks:

- All `expected_prompt_logits` values, position by position.
- All `expected_step_logits` values, step by step.
- Every generated token against `expected_tokens`.
- The complete CPU string against the complete GPU string.

Each comparison counts values outside `1e-9` and tracks the maximum absolute
difference. Any value outside the tolerance, any token mismatch, or any CPU/GPU
text difference returns exit code 1. The output names the failing path and
prints the first few mismatches.

## Performance

- Mount time is excluded from both timings.
- CPU: one warm-up generation, then a timed generation; report total and
  milliseconds per token.
- GPU: one warm-up generation, then a timed generation with `gpu.sync` before
  stopping the clock; report total and milliseconds per token.
- Also report a single forward pass over the final full context for each path
  (the prompt plus generated tokens, 116 tokens) so throughput is visible
  separately from the autoregressive loop.
- Report the speedup as CPU time divided by GPU time. If the GPU is slower for
  the one-token-at-a-time loop because of launch overhead, print that plainly.
- Use `time.monotonicNanos` for both paths.

## Files and documentation

New example:

```text
examples/projects/tiny_transformer/
  README.md
  train.py
  main.tx
  model.txm
```

Changes elsewhere:

- `Makefile`: `transformer` runs the example; `transformer-train` runs the
  exporter with the venv Python when provided.
- `.gitignore`: ignore `data/` and `checkpoint.pt`; keep `model.txm` tracked.
- `website/content/docs/examples/tiny-transformer.md`: architecture, validation,
  sample output, CPU/GPU numbers, reproduction commands.
- `website/content/docs/examples/_index.md`: link the new example.
- `website/content/docs/stdlib/math.md` and the stdlib index: document the new
  module.
- Compiler tests for host math and GPU `sqrt`.

## Error handling

- Missing `model.txm`: `gpu.mount` already fails with a runtime error.
- Malformed `config`, vocabulary length mismatch, or prompt token mismatch:
  print a specific message and return 1 before inference.
- Validation mismatch: print max difference, count outside tolerance, and the
  first mismatching token index; return 1.
- Training download failure: print the failing URL and suggest `--data`.
- Export with an unstable greedy margin: fail before writing the TXM.
- No CUDA device: `gpu.mount` fails; the README states that the current TXM
  reader requires the CUDA build, as in the MNIST benchmark.

## Testing

- `go test` for the standard library and codegen packages touched by the math
  module.
- `go test` for GPU `sqrt` codegen.
- `python3 train.py --sample` is not planned; training is required to produce a
  real model. The committed TXM is the reproducible inference fixture.
- `txm.verify` on the exported file.
- Run the Trux example and require zero validation failures.
- Build the docs and run the existing markdown output check.
- Confirm `make transformer` works after moving the venv out of the way, proving
  the runtime path needs no Python.

## Risks and trade-offs

- A 2-layer 128-wide character model will produce rough, sometimes repetitive
  Shakespeare-like text. That is expected for this size.
- Greedy decoding is chosen for exact comparison even though sampling would look
  more natural.
- The CPU path needs the CUDA build because `gpu.mount` is the only TXM reader.
  This matches the existing MNIST CPU/GPU benchmark.
- FP64 GPU math is slower on the GTX 1660, but it lets the CPU, GPU, and Python
  paths agree to a tight tolerance.
- The committed model is about 3.5 MB of binary data. The training script can
  regenerate it, and the file stays well below normal repository limits.

## Open questions

None.
