# Tiny transformer

A two-layer character-level transformer trained in PyTorch, exported to a TXM1
file, and run in Trux. Training stays in Python; `main.tx` only loads the
weights, runs the forward pass, and compares its output with the reference.

The model is deliberately small enough to keep entirely resident: 65
characters, a 128-token context, 128-wide embeddings, four heads, and a
512-wide MLP. It has 429,889 parameters, or about 3.5 MB as float64. The
committed `model.txm` also carries the prompt tokens and every expected logit,
so `make transformer` does not need Python or PyTorch.

Running either the CPU or GPU path needs an NVIDIA GPU and `nvcc`, because
`gpu.mount` is the only TXM reader in Trux. The CPU section launches no
kernels. Python is only needed to retrain or re-export.

## Run

```console
$ make transformer                                           # default prompt ("JULIET:\n", 108 tokens)
$ make transformer PROMPT="what is your name?\n" TOKENS=60   # custom prompt and token count
$ make transformer-train                                     # optional: train and export again
```

`make transformer` builds the compiler and runs `main.tx`. You can pass
`PROMPT` and `TOKENS` on the command line (or edit `prompt` and `genLen` in
`main.tx`); when they differ from the committed `"JULIET:\n"` fixture, the GPU
path is checked value-by-value (`1e-9`) and token-by-token against the CPU
path. To train from scratch, `train.py` needs `numpy` and `torch`; it downloads
Tiny Shakespeare (about 1.1 MB) into `data/` on the first run, trains 8,000
steps, and writes `checkpoint.pt` and `model.txm`. `data/` and `checkpoint.pt`
are gitignored; `model.txm` is committed so the inference example runs without
a Python environment.

To export from an existing checkpoint without training:

```console
$ cd examples/projects/tiny_transformer
$ python3 train.py --export-only
```

## Output

This is one real run on a GTX 1660 with CUDA 12.0. The generated text is
greedy, so it is exactly the token sequence the Python reference produced.

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
```

The check is not just the argmax. The prompt is 8 characters, so its forward
pass produces `8 * 65 = 520` final-layer values. Each of the 108 generated
tokens has a distribution of 65 values, for `108 * 65 = 7,020` more. That is
7,540 values per path, and every one is compared with PyTorch's float64
reference at an absolute tolerance of `1e-9`. The generated token IDs and the
whole prompt-to-text path are compared exactly.

The largest differences are `7.4e-14` on the CPU and `6.4e-14` on the GPU,
which is float64 rounding from a different summation order. The smallest
top-2 margin at any greedy step is `0.00579486`, about eleven orders of
magnitude above those differences, so no token can flip on rounding.

## CPU vs GPU

The program warms both paths, then times one generation of 108 tokens and one
full-context prefill of 116 tokens. The CPU path uses scalar host loops; the GPU
path uses 25 double-precision kernel launches per forward pass, a `gpu.sync`
after each step, and a copy of the logits back to the host for argmax. There is
no KV cache: every step recomputes the full context, which keeps the CPU and
GPU arithmetic easy to compare.

One representative run:

| Pass | CPU | GPU | CPU / GPU |
| --- | ---: | ---: | ---: |
| 108-token generation | 3.478 s | 0.3553 s | 9.78x |
| per generated token | 32.2 ms | 3.29 ms | 9.78x |
| 116-token prefill | 59.7 ms | 4.10 ms | 14.54x |

Three runs on the same machine gave CPU generation `3.32` to `3.48` s and GPU
generation `0.349` to `0.359` s, or `9.5x` to `9.8x`. Prefill is a single pass
and is noisier: CPU `59.7` to `69.2` ms, GPU `3.88` to `5.59` ms, or `10.7x` to
`17.8x`. Run-to-run variation is largest for the prefill. The last column is
CPU time divided by GPU time, so values above 1 mean the GPU is faster.

The speedup is modest for a GPU because the model is small and the weights are
float64. A consumer GTX 1660 runs double precision at a fraction of its float32
rate. Per-step launch overhead, the sync, and the copy also matter when each
forward pass is only a few milliseconds. A KV cache and an on-device argmax
would reduce the 25 launches and the copy, but they would make the CPU/GPU
comparison less direct; this example favors the exact check.

## Tensor layout

All layer weights have a leading layer dimension, so the CPU loop indexes
`layer * ...` and the kernels receive a `weightBase` offset into one flat
buffer. Linear weights are stored `[out, in]`, the same layout as PyTorch's
`nn.Linear.weight`, so each row of the weight and each row of the input are
walked in the same order. The `c_attn` weight is split into `wq`, `wk`, and
`wv` at export.

| Name | Shape | Dtype | Meaning |
| --- | --- | --- | --- |
| `config` | `[6]` | int | vocab, context, layers, heads, width, MLP |
| `inv_width` | `[1]` | float | `1 / width` for layer norm |
| `attn_scale` | `[1]` | float | `1 / sqrt(head_dim)` |
| `ln_eps` | `[1]` | float | layer norm epsilon, `1e-5` |
| `tok_emb` | `[65, 128]` | float | character embeddings |
| `pos_emb` | `[128, 128]` | float | position embeddings |
| `ln1_g`, `ln1_b` | `[2, 128]` | float | first layer norm per block |
| `wq`, `wk`, `wv`, `wo` | `[2, 128, 128]` | float | attention projections |
| `bq`, `bk`, `bv`, `bo` | `[2, 128]` | float | attention biases |
| `ln2_g`, `ln2_b` | `[2, 128]` | float | second layer norm per block |
| `w1` | `[2, 512, 128]` | float | MLP up projection |
| `b1` | `[2, 512]` | float | MLP up bias |
| `w2` | `[2, 128, 512]` | float | MLP down projection |
| `b2` | `[2, 128]` | float | MLP down bias |
| `lnf_g`, `lnf_b` | `[128]` | float | final layer norm |
| `wout` | `[65, 128]` | float | output projection |
| `bout` | `[65]` | float | output bias |
| `prompt_tokens` | `[8]` | int | `JULIET:\n` in vocab IDs |
| `expected_prompt_logits` | `[8, 65]` | float | PyTorch logits for all prompt positions |
| `expected_step_logits` | `[108, 65]` | float | PyTorch logits for each greedy step |
| `expected_tokens` | `[108]` | int | greedy token IDs to compare |

The prompt is 8 characters and the generation is 108 tokens, so the longest
forward pass is 116 tokens, inside the 128-token context. The vocabulary is the
65 distinct characters in Tiny Shakespeare:

```text
\n !$&',-.3:;?ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz
```

## How the chain is checked

1. `train.py` trains in PyTorch, loads the best validation checkpoint, converts
   the model to float64, and recomputes the prompt logits, every step's logits,
   and the greedy token IDs.
2. It refuses to export if any greedy step has a top-2 margin at or below
   `1e-7`. The smallest margin in the committed run is `0.00579486`.
3. It writes the weights and fixtures with `python/txm.py` and runs
   `txm.verify` on the file.
4. `main.tx` mounts the file once, runs the same forward pass on the host and
   with CUDA kernels, compares all 7,540 values per path, then compares the
   generated text with the reference.

The host math calls live in `hostmath.tx`. A `math` import is visible to the
whole file, including kernels, where bare `sqrt` and `exp` are the GPU
builtins. Keeping the two host wrappers in a separate module avoids shadowing
those kernel names.

## Training

The defaults are in `train.py`: batch 64, context 128, 2 layers, 4 heads,
128-wide, AdamW with a peak learning rate of `3e-4`, 100 warmup steps, cosine
decay, and 8,000 steps. The best validation loss was `1.359375`. Training and
export are Python-only; nothing in `main.tx` updates a weight.

The model is a language model, not a chatbot. It was trained on about 1.1 MB of
Shakespeare and produces plausible character sequences, not facts. The point is
that a real transformer's entire inference path matches the Python reference
value by value on both CPU and GPU.

## Files

| File | |
| --- | --- |
| `train.py` | train in PyTorch, build the float64 reference, write TXM1 |
| `main.tx` | CPU and GPU forward passes, value checks, greedy generation, timings |
| `hostmath.tx` | host `hexp` and `hsqrt` wrappers around `math` |
| `model.txm` | committed inference artifact, weights plus fixtures |
| `README.md` | these notes |

`model.txm` is 3,501,437 bytes: 30 tensors, including the weights, config, and
the reference logits and tokens. The training data and checkpoint are not part
of the repository.
