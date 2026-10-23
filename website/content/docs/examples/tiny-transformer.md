---
title: "Tiny transformer"
description: "Train a character-level transformer in PyTorch, run it on CPU and GPU in Trux, and check every final-layer value against Python."
icon: "psychology"
weight: 85
---

This is a two-layer, four-head character-level transformer trained on Tiny Shakespeare in PyTorch, exported to a
TXM1 file, and run in Trux. Training is Python-only; the Trux program loads the weights, runs the forward pass on the
host and on the GPU, checks every final-layer value against PyTorch's float64 reference, and compares the generated
text.

The model is small enough to stay resident in both host and device memory: 65 characters, a 128-token context,
128-wide embeddings, four attention heads, and a 512-wide MLP. That is 429,889 parameters, or about 3.5 MB in float64.
The committed `model.txm` carries the weights, the prompt tokens, the expected logits, and the expected greedy token
IDs, so `make transformer` runs without Python or PyTorch.

## The check

Checking only the generated text is too weak. A wrong attention scale, an off-by-one causal mask, or a residual added
in the wrong place can still leave the top token unchanged for several steps while corrupting the distribution.

This example checks the distributions first. The prompt is `"JULIET:\n"` (8 characters), so the prompt pass produces
`8 * 65 = 520` logits. The program then generates 108 tokens greedily, checking all 65 logits at every step before
taking the argmax (`108 * 65 = 7,020` more values). Both the CPU path and the GPU path compare all 7,540 values against
PyTorch at an absolute tolerance of `1e-9`, check every chosen token ID, and compare the decoded text.

## The model and the file

`train.py` trains `TinyGPT` on Tiny Shakespeare with AdamW (`3e-4` peak learning rate, 100 warmup steps, cosine decay,
batch size 64, context 128) for 8,000 steps. The committed model reached a validation loss of `1.359375`. At export
time, the script loads the best checkpoint, converts the model to float64 on the CPU, and records the prompt logits,
step logits, and greedy tokens from those exact float64 weights. The smallest top-2 logit margin across all 108 steps
is `0.00579486`, so no step is close to a tie.

All per-layer tensors have a leading layer dimension (`[2, ...]`). The CPU loop indexes `layer * ...` and the CUDA
kernels take a `weightBase` offset into the same flat buffers. Linear weights are stored `[out, in]`, matching
PyTorch's `nn.Linear.weight`, and the combined `c_attn` projection is split into `wq`, `wk`, and `wv` at export.

| Name | Shape | Dtype | Meaning |
| --- | --- | --- | --- |
| `config` | `[6]` | int | vocab size, context length, layers, heads, width, MLP width |
| `inv_width` | `[1]` | float | `1 / width` for layer norm |
| `attn_scale` | `[1]` | float | `1 / sqrt(head_dim)` |
| `ln_eps` | `[1]` | float | layer norm epsilon, `1e-5` |
| `tok_emb` | `[65, 128]` | float | character embeddings |
| `pos_emb` | `[128, 128]` | float | position embeddings |
| `ln1_g`, `ln1_b` | `[2, 128]` | float | pre-attention layer norm |
| `wq`, `wk`, `wv`, `wo` | `[2, 128, 128]` | float | attention projections |
| `bq`, `bk`, `bv`, `bo` | `[2, 128]` | float | attention biases |
| `ln2_g`, `ln2_b` | `[2, 128]` | float | pre-MLP layer norm |
| `w1` | `[2, 512, 128]` | float | MLP up projection (ReLU) |
| `b1` | `[2, 512]` | float | MLP up bias |
| `w2` | `[2, 128, 512]` | float | MLP down projection |
| `b2` | `[2, 128]` | float | MLP down bias |
| `lnf_g`, `lnf_b` | `[128]` | float | final layer norm |
| `wout` | `[65, 128]` | float | output head |
| `bout` | `[65]` | float | output bias |
| `prompt_tokens` | `[8]` | int | `"JULIET:\n"` in vocabulary IDs |
| `expected_prompt_logits` | `[8, 65]` | float | PyTorch logits for every prompt position |
| `expected_step_logits` | `[108, 65]` | float | PyTorch logits for each greedy step |
| `expected_tokens` | `[108]` | int | greedy token IDs from the PyTorch check |

## The CPU and GPU passes

`main.tx` mounts `model.txm` once, reads the configuration with `gpu.hostInts` and `gpu.dim`, and keeps both the host
slices and the device buffers resident for the whole run. Neither path uses a KV cache: each step runs the full
context of length `n` (`8` up to `115`), which keeps the CPU loops and the GPU kernels doing the same arithmetic in
the same order.

On the GPU, one forward pass is 25 `gpu.launch1d` calls:

1. `embedK` adds `tok_emb[token]` and `pos_emb[t]` for every position `t < n`.
2. In each of the two blocks, `layerNormK` normalizes each row, three `linearK` launches compute `q`, `k`, and `v`,
   `scoresK` computes the lower-triangular causal dot products scaled by `1 / sqrt(head_dim)`, `softmaxK` normalizes
   each causal row, `weightedK` mixes `v`, and `linearAddK` projects with `wo` and adds the residual into `x`.
3. A second `layerNormK`, a ReLU `linearK` up to 512, and a `linearAddK` down to 128 run the MLP and its residual.
4. One final `layerNormK` and `linearK` produce the `[n, 65]` logits.

The host math calls (`math.exp` and `math.sqrt`) live in a tiny companion module, `hostmath.tx`. An import applies to
the whole file, including its kernels, where bare `exp` and `sqrt` are the CUDA builtins; keeping `import "math"` in
`hostmath.tx` avoids shadowing the kernel builtins.

{{< details summary="`examples/projects/tiny_transformer/main.tx`" >}}

{{< include "examples/projects/tiny_transformer/main.tx" "trux" >}}

{{< /details >}}

{{< details summary="`examples/projects/tiny_transformer/hostmath.tx`" >}}

{{< include "examples/projects/tiny_transformer/hostmath.tx" "trux" >}}

{{< /details >}}

## Run it

From the repository root, after [building `trux`](/docs/getting-started/install/):

```bash
make transformer
make transformer PROMPT="what is your name?\n" TOKENS=60
```

That uses the committed `model.txm` and needs an NVIDIA GPU and `nvcc` (because `gpu.mount` is the TXM reader), but no
Python. When `PROMPT` or `TOKENS` differs from the committed `"JULIET:\n"` fixture, the GPU run is still checked value
by value (`1e-9`) and token by token against the CPU run.

To retrain from scratch or re-export from an existing checkpoint:

```bash
make transformer-train
# or, inside examples/projects/tiny_transformer:
python3 train.py --export-only
```

The trainer needs `numpy` and `torch`. Its first run downloads Tiny Shakespeare (about 1.1 MB) into `data/`.

## Output

This is a real run on an AMD Ryzen 9 5900X and a GTX 1660 with CUDA 12.0:

```text
model: layers 2 heads 4 width 128 mlp 512 vocab 65 context 128
prompt:
JULIET:

reference text:
JULIET:
I will stand the state of the dead of the death.

CAPULET:
And there is the state of the death of the death.
cpu text:
JULIET:
I will stand the state of the dead of the death.

CAPULET:
And there is the state of the death of the death.
gpu text:
JULIET:
I will stand the state of the dead of the death.

CAPULET:
And there is the state of the death of the death.

cpu values checked 7540
cpu values outside 1e-9 0
cpu max logit difference 7.41628980449605e-14
cpu token mismatches 0
cpu prompt token mismatches 0
gpu values checked 7540
gpu values outside 1e-9 0
gpu max logit difference 6.3504757008559e-14
gpu token mismatches 0
gpu prompt token mismatches 0
text mismatches 0

cpu generation nanos 3478436676
gpu generation nanos 355320120
cpu nanoseconds per token 32207747
gpu nanoseconds per token 3290001
cpu prefill nanos 59667451
gpu prefill nanos 4101356
cpu/gpu generation 9.78x
cpu/gpu prefill 14.54x
```

The largest difference across all 7,540 logits is `7.42e-14` on the CPU and `6.35e-14` on the GPU, which is ordinary
float64 rounding between PyTorch's kernels and Trux's scalar loops. Because the smallest top-2 margin is `0.00579486`
(about `8e10` times larger), greedy decoding is deterministic across all three runs. The prose is repetitive because
the model is tiny and decoding is greedy; what matters here is that Python, Trux CPU, and Trux GPU produce the exact
same sequence from the exact same weights.

## CPU vs GPU

Both paths run one warm-up generation pass before the timer starts, then time the 108-token autoregressive loop and a
single 116-token prefill forward pass:

| Pass | CPU | GPU | CPU / GPU |
| --- | ---: | ---: | ---: |
| 108-token generation | 3.478 s | 0.3553 s | 9.78x |
| per generated token | 32.2 ms | 3.29 ms | 9.78x |
| 116-token prefill | 59.7 ms | 4.10 ms | 14.54x |

Across three runs on the same machine, CPU generation was `3.32` to `3.48` s and GPU generation was `0.349` to `0.359` s
(`9.5x` to `9.8x`). Single-pass prefill is shorter and noisier: `59.7` to `69.2` ms on the CPU and `3.88` to `5.59` ms
on the GPU (`10.7x` to `17.8x`).

The GPU speedup is around 10x rather than 100x for three reasons:

1. **Float64 tensors.** TXM float tensors are double precision, and a consumer GTX 1660 runs FP64 at 1/32 of its FP32
   throughput.
2. **Small matrix sizes.** At width 128 and context 8 to 115, each launch has only a few thousand threads.
3. **No KV cache and per-step host argmax.** Every generated token launches 25 kernels, calls `gpu.sync`, and copies
   the logits back to the host. That keeps every step's full logit vector easy to validate against PyTorch.

{{< details summary="`examples/projects/tiny_transformer/train.py`" >}}

{{< include "examples/projects/tiny_transformer/train.py" "python" >}}

{{< /details >}}

## Files

| File | |
| --- | --- |
| `main.tx` | CPU and GPU transformer forward passes, value checks, greedy generation, and timings |
| `hostmath.tx` | host `hexp` and `hsqrt` wrappers around `math` so kernel builtins are not shadowed |
| `train.py` | train in PyTorch, verify the float64 reference, and write `model.txm` |
| `model.txm` | committed 3.5 MB TXM1 artifact: config, weights, prompt, expected logits, and expected tokens |
| `README.md` | repository notes |

## Related

- [MNIST MLP](/docs/examples/mnist-mlp/): the smaller three-layer classification example that uses the same TXM1
  verification pattern.
- [`math` reference](/docs/stdlib/math/): `math.exp` and `math.sqrt` on the host, plus `exp` and `sqrt` in kernels.
- [`gpu` reference](/docs/stdlib/gpu/): device buffers, launches, and `gpu.mount`.
- [TXM1 files](/docs/design-notes/txm/): the binary tensor container format.
