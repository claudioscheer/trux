#!/usr/bin/env python3
"""Train the tiny character-level transformer used by the Trux example.

The model is deliberately small: two pre-layer-norm transformer blocks, four
attention heads, 128-wide embeddings, and a 128-token context. It is trained on
Tiny Shakespeare, then exported as float64 weights together with the exact
prompt logits and greedy token IDs that main.tx checks.

    python3 train.py
    python3 train.py --export-only
    python3 train.py --steps 12000 --prompt $'JULIET:\n'

The first run downloads Tiny Shakespeare (about 1.1 MB) into data/. Pass
--data PATH to use a local text file instead. The training checkpoint is
checkpoint.pt; the inference artifact is model.txm. Only model.txm is meant to
be committed.
"""

from __future__ import annotations

import argparse
import math
import sys
import time
import urllib.request
from pathlib import Path

import numpy as np
import torch
import torch.nn as nn
import torch.nn.functional as F

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / "python"))

import txm

DATA_URL = (
    "https://raw.githubusercontent.com/karpathy/char-rnn/master/data/"
    "tinyshakespeare/input.txt"
)
HERE = Path(__file__).resolve().parent
DEFAULT_DATA = HERE / "data" / "tinyshakespeare.txt"
DEFAULT_OUTPUT = HERE / "model.txm"
DEFAULT_CHECKPOINT = HERE / "checkpoint.pt"
DEFAULT_PROMPT = "JULIET:\n"
DEFAULT_TOKENS = 108


class CausalSelfAttention(nn.Module):
    def __init__(self, n_embd: int, n_head: int):
        super().__init__()
        if n_embd % n_head != 0:
            raise ValueError("n_embd must be divisible by n_head")
        self.n_head = n_head
        self.head_dim = n_embd // n_head
        self.c_attn = nn.Linear(n_embd, 3 * n_embd)
        self.c_proj = nn.Linear(n_embd, n_embd)

    def forward(self, x: torch.Tensor, mask: torch.Tensor) -> torch.Tensor:
        batch, length, width = x.shape
        qkv = self.c_attn(x)
        q, k, v = qkv.split(width, dim=2)
        q = q.view(batch, length, self.n_head, self.head_dim).transpose(1, 2)
        k = k.view(batch, length, self.n_head, self.head_dim).transpose(1, 2)
        v = v.view(batch, length, self.n_head, self.head_dim).transpose(1, 2)

        scores = (q @ k.transpose(-2, -1)) * (1.0 / math.sqrt(self.head_dim))
        scores = scores.masked_fill(mask[:, :, :length, :length] == 0, float("-inf"))
        attention = F.softmax(scores, dim=-1)
        out = attention @ v
        out = out.transpose(1, 2).contiguous().view(batch, length, width)
        return self.c_proj(out)


class Block(nn.Module):
    def __init__(self, n_embd: int, n_head: int):
        super().__init__()
        self.ln1 = nn.LayerNorm(n_embd)
        self.attn = CausalSelfAttention(n_embd, n_head)
        self.ln2 = nn.LayerNorm(n_embd)
        self.mlp = nn.Sequential(
            nn.Linear(n_embd, 4 * n_embd),
            nn.ReLU(),
            nn.Linear(4 * n_embd, n_embd),
        )

    def forward(self, x: torch.Tensor, mask: torch.Tensor) -> torch.Tensor:
        x = x + self.attn(self.ln1(x), mask)
        x = x + self.mlp(self.ln2(x))
        return x


class TinyGPT(nn.Module):
    def __init__(
        self,
        vocab_size: int,
        block_size: int,
        n_layer: int,
        n_head: int,
        n_embd: int,
    ):
        super().__init__()
        self.block_size = block_size
        self.tok_emb = nn.Embedding(vocab_size, n_embd)
        self.pos_emb = nn.Embedding(block_size, n_embd)
        self.blocks = nn.ModuleList(
            [Block(n_embd, n_head) for _ in range(n_layer)]
        )
        self.ln_f = nn.LayerNorm(n_embd)
        self.head = nn.Linear(n_embd, vocab_size)
        self.register_buffer(
            "causal_mask",
            torch.tril(torch.ones(block_size, block_size, dtype=torch.bool)).view(
                1, 1, block_size, block_size
            ),
            persistent=False,
        )
        self.apply(self._init_weights)

    def _init_weights(self, module: nn.Module) -> None:
        if isinstance(module, nn.Linear):
            nn.init.normal_(module.weight, mean=0.0, std=0.02)
            if module.bias is not None:
                nn.init.zeros_(module.bias)
        elif isinstance(module, nn.Embedding):
            nn.init.normal_(module.weight, mean=0.0, std=0.02)

    def forward(
        self, idx: torch.Tensor, targets: torch.Tensor | None = None
    ) -> tuple[torch.Tensor, torch.Tensor | None]:
        batch, length = idx.shape
        if length > self.block_size:
            raise ValueError("sequence is longer than block_size")
        positions = torch.arange(length, device=idx.device)
        x = self.tok_emb(idx) + self.pos_emb(positions)
        for block in self.blocks:
            x = block(x, self.causal_mask)
        x = self.ln_f(x)
        logits = self.head(x)
        loss = None
        if targets is not None:
            loss = F.cross_entropy(
                logits.reshape(-1, logits.size(-1)), targets.reshape(-1)
            )
        return logits, loss


def load_text(data_path: Path | None) -> str:
    if data_path is not None:
        return data_path.read_text(encoding="utf-8")
    if not DEFAULT_DATA.exists():
        DEFAULT_DATA.parent.mkdir(parents=True, exist_ok=True)
        print("downloading Tiny Shakespeare from %s" % DATA_URL, flush=True)
        try:
            with urllib.request.urlopen(DATA_URL, timeout=60) as response:
                payload = response.read()
        except OSError as err:
            raise SystemExit(
                "could not download %s: %s\nPass --data PATH with a local text file."
                % (DATA_URL, err)
            ) from None
        DEFAULT_DATA.write_bytes(payload)
        print("wrote %s (%d bytes)" % (DEFAULT_DATA, len(payload)), flush=True)
    return DEFAULT_DATA.read_text(encoding="utf-8")


def build_vocab(text: str) -> tuple[list[str], dict[str, int]]:
    chars = sorted(set(text))
    return chars, {ch: index for index, ch in enumerate(chars)}


def encode(text: str, stoi: dict[str, int]) -> list[int]:
    try:
        return [stoi[ch] for ch in text]
    except KeyError as err:
        raise SystemExit("prompt contains a character outside the vocabulary: %r" % err) from None


def decode(ids, chars: list[str]) -> str:
    return "".join(chars[index] for index in ids)


def tensor_literal(chars: list[str]) -> str:
    """The vocabulary as a Trux string literal."""
    out = []
    for ch in chars:
        if ch == "\\":
            out.append("\\\\")
        elif ch == '"':
            out.append('\\"')
        elif ch == "\n":
            out.append("\\n")
        elif ch == "\t":
            out.append("\\t")
        elif ch == "\r":
            out.append("\\r")
        else:
            out.append(ch)
    return '"' + "".join(out) + '"'


def get_batch(
    data: torch.Tensor,
    batch_size: int,
    block_size: int,
    device: torch.device,
) -> tuple[torch.Tensor, torch.Tensor]:
    if len(data) <= block_size + 1:
        raise SystemExit("training split is too short for the requested block size")
    starts = torch.randint(0, len(data) - block_size - 1, (batch_size,))
    x = torch.stack([data[start : start + block_size] for start in starts])
    y = torch.stack([data[start + 1 : start + block_size + 1] for start in starts])
    return x.to(device), y.to(device)


def fixed_validation_batches(
    data: torch.Tensor, block_size: int, count: int
) -> tuple[torch.Tensor, torch.Tensor]:
    if len(data) < block_size * count + 1:
        raise SystemExit("validation split is too short for the requested evaluation")
    starts = [index * block_size for index in range(count)]
    x = torch.stack([data[start : start + block_size] for start in starts])
    y = torch.stack([data[start + 1 : start + block_size + 1] for start in starts])
    return x, y


def learning_rate(step: int, max_steps: int, peak: float, warmup: int) -> float:
    if step < warmup:
        return peak * (step + 1) / max(warmup, 1)
    ratio = (step - warmup) / max(max_steps - warmup, 1)
    ratio = min(max(ratio, 0.0), 1.0)
    cosine = 0.5 * (1.0 + math.cos(math.pi * ratio))
    return peak * (0.1 + 0.9 * cosine)


def estimate_loss(
    model: TinyGPT,
    val_x: torch.Tensor,
    val_y: torch.Tensor,
    batch_size: int,
) -> float:
    model.eval()
    total = 0.0
    batches = 0
    with torch.no_grad():
        for start in range(0, len(val_x), batch_size):
            x = val_x[start : start + batch_size]
            y = val_y[start : start + batch_size]
            _, loss = model(x, y)
            total += float(loss.item())
            batches += 1
    model.train()
    return total / max(batches, 1)


@torch.no_grad()
def sample_text(
    model: TinyGPT,
    prompt_ids: list[int],
    tokens: int,
    chars: list[str],
    temperature: float = 0.8,
    top_k: int = 40,
) -> str:
    model.eval()
    idx = torch.tensor([prompt_ids], dtype=torch.long, device=next(model.parameters()).device)
    generated = []
    for _ in range(tokens):
        logits, _ = model(idx)
        logits = logits[0, -1] / max(temperature, 1e-6)
        if top_k > 0:
            values, _ = torch.topk(logits, min(top_k, logits.size(-1)))
            cutoff = values[-1]
            logits = torch.where(logits < cutoff, torch.full_like(logits, float("-inf")), logits)
        probabilities = F.softmax(logits, dim=-1)
        next_id = int(torch.multinomial(probabilities, 1).item())
        generated.append(next_id)
        idx = torch.cat(
            [idx, torch.tensor([[next_id]], dtype=torch.long, device=idx.device)],
            dim=1,
        )
    model.train()
    return decode(generated, chars)


def load_checkpoint(path: Path, device: torch.device) -> dict:
    if not path.exists():
        raise SystemExit("checkpoint not found at %s (use --checkpoint PATH)" % path)
    return torch.load(path, map_location=device, weights_only=False)


def save_checkpoint(
    path: Path,
    model: TinyGPT,
    optimizer: torch.optim.Optimizer,
    step: int,
    best_val: float,
    best_state: dict | None,
    config: dict,
    chars: list[str],
) -> None:
    torch.save(
        {
            "model": model.state_dict(),
            "optimizer": optimizer.state_dict(),
            "step": step,
            "best_val": best_val,
            "best_state": best_state,
            "config": config,
            "chars": chars,
        },
        path,
    )


def reference_export(
    model: TinyGPT,
    prompt: str,
    tokens: int,
    chars: list[str],
    stoi: dict[str, int],
    config: dict,
    output: Path,
) -> None:
    model.cpu().double().eval()
    prompt_ids = encode(prompt, stoi)
    if len(prompt_ids) + tokens > model.block_size:
        raise SystemExit(
            "prompt plus generated tokens is %d, above block size %d"
            % (len(prompt_ids) + tokens, model.block_size)
        )

    with torch.no_grad():
        prompt_tensor = torch.tensor([prompt_ids], dtype=torch.long)
        logits, _ = model(prompt_tensor)
        prompt_logits = logits[0].numpy().astype(np.float64, copy=True)

        context = list(prompt_ids)
        generated = []
        step_logits = []
        smallest_margin = float("inf")

        for step in range(tokens):
            context_tensor = torch.tensor([context], dtype=torch.long)
            logits, _ = model(context_tensor)
            last = logits[0, -1]
            ranked = torch.topk(last, 2).values
            margin = float((ranked[0] - ranked[1]).item())
            smallest_margin = min(smallest_margin, margin)
            if margin <= 1e-7:
                raise SystemExit(
                    "greedy margin at step %d is %.3g; choose another prompt "
                    "or train longer" % (step, margin)
                )
            next_id = int(torch.argmax(last).item())
            step_logits.append(last.numpy().astype(np.float64, copy=True))
            generated.append(next_id)
            context.append(next_id)

    generated_array = np.asarray(generated, dtype=np.int64)
    step_logits_array = np.stack(step_logits, axis=0)

    state = model.state_dict()

    def value(name: str) -> np.ndarray:
        return state[name].detach().cpu().numpy().astype(np.float64, copy=True)

    layers = config["n_layer"]
    width = config["n_embd"]
    hidden = 4 * width

    wq, wk, wv = [], [], []
    bq, bk, bv = [], [], []
    ln1_g, ln1_b = [], []
    wo, bo = [], []
    ln2_g, ln2_b = [], []
    w1, b1 = [], []
    w2, b2 = [], []

    for layer in range(layers):
        prefix = "blocks.%d." % layer
        qkv_weight = value(prefix + "attn.c_attn.weight")
        qkv_bias = value(prefix + "attn.c_attn.bias")
        q, k, v = np.split(qkv_weight, 3, axis=0)
        qb, kb, vb = np.split(qkv_bias, 3, axis=0)
        wq.append(q)
        wk.append(k)
        wv.append(v)
        bq.append(qb)
        bk.append(kb)
        bv.append(vb)
        ln1_g.append(value(prefix + "ln1.weight"))
        ln1_b.append(value(prefix + "ln1.bias"))
        wo.append(value(prefix + "attn.c_proj.weight"))
        bo.append(value(prefix + "attn.c_proj.bias"))
        ln2_g.append(value(prefix + "ln2.weight"))
        ln2_b.append(value(prefix + "ln2.bias"))
        w1.append(value(prefix + "mlp.0.weight"))
        b1.append(value(prefix + "mlp.0.bias"))
        w2.append(value(prefix + "mlp.2.weight"))
        b2.append(value(prefix + "mlp.2.bias"))

    tensors = [
        txm.ints(
            "config",
            np.asarray(
                [
                    config["vocab_size"],
                    config["block_size"],
                    config["n_layer"],
                    config["n_head"],
                    config["n_embd"],
                    hidden,
                ],
                dtype=np.int64,
            ),
        ),
        txm.floats("inv_width", np.asarray([1.0 / width], dtype=np.float64)),
        txm.floats(
            "attn_scale",
            np.asarray(
                [1.0 / math.sqrt(width // config["n_head"])], dtype=np.float64
            ),
        ),
        txm.floats("ln_eps", np.asarray([1e-5], dtype=np.float64)),
        txm.floats("tok_emb", value("tok_emb.weight")),
        txm.floats("pos_emb", value("pos_emb.weight")),
        txm.floats("ln1_g", np.stack(ln1_g).reshape(layers, width), (layers, width)),
        txm.floats("ln1_b", np.stack(ln1_b).reshape(layers, width), (layers, width)),
        txm.floats("wq", np.stack(wq).reshape(layers, width, width), (layers, width, width)),
        txm.floats("bq", np.stack(bq).reshape(layers, width), (layers, width)),
        txm.floats("wk", np.stack(wk).reshape(layers, width, width), (layers, width, width)),
        txm.floats("bk", np.stack(bk).reshape(layers, width), (layers, width)),
        txm.floats("wv", np.stack(wv).reshape(layers, width, width), (layers, width, width)),
        txm.floats("bv", np.stack(bv).reshape(layers, width), (layers, width)),
        txm.floats("wo", np.stack(wo).reshape(layers, width, width), (layers, width, width)),
        txm.floats("bo", np.stack(bo).reshape(layers, width), (layers, width)),
        txm.floats("ln2_g", np.stack(ln2_g).reshape(layers, width), (layers, width)),
        txm.floats("ln2_b", np.stack(ln2_b).reshape(layers, width), (layers, width)),
        txm.floats("w1", np.stack(w1).reshape(layers, hidden, width), (layers, hidden, width)),
        txm.floats("b1", np.stack(b1).reshape(layers, hidden), (layers, hidden)),
        txm.floats("w2", np.stack(w2).reshape(layers, width, hidden), (layers, width, hidden)),
        txm.floats("b2", np.stack(b2).reshape(layers, width), (layers, width)),
        txm.floats("lnf_g", value("ln_f.weight")),
        txm.floats("lnf_b", value("ln_f.bias")),
        txm.floats("wout", value("head.weight")),
        txm.floats("bout", value("head.bias")),
        txm.ints("prompt_tokens", np.asarray(prompt_ids, dtype=np.int64)),
        txm.floats("expected_prompt_logits", prompt_logits, (len(prompt_ids), config["vocab_size"])),
        txm.floats("expected_step_logits", step_logits_array, (tokens, config["vocab_size"])),
        txm.ints("expected_tokens", generated_array),
    ]
    for tensor in tensors:
        if tensor.name == "":
            raise SystemExit("internal error: unnamed tensor")
    txm.pack(str(output), tensors)

    expected = {tensor.name: tensor.dims for tensor in tensors}
    problems = txm.verify(str(output), expected)
    if problems:
        raise SystemExit("TXM verification failed:\n" + "\n".join(problems))

    text = prompt + decode(generated, chars)
    print("")
    print("prompt:")
    print(prompt)
    print("generated:")
    print(decode(generated, chars))
    print("reference text:")
    print(text)
    print("smallest top-2 margin: %.6g" % smallest_margin)
    print("vocab (%d): %s" % (len(chars), tensor_literal(chars)))
    print(txm.info(str(output)), flush=True)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--data", type=Path, default=None, help="local training text")
    parser.add_argument("--output", type=Path, default=DEFAULT_OUTPUT)
    parser.add_argument("--checkpoint", type=Path, default=DEFAULT_CHECKPOINT)
    parser.add_argument("--steps", type=int, default=8000)
    parser.add_argument("--batch-size", type=int, default=64)
    parser.add_argument("--block-size", type=int, default=128)
    parser.add_argument("--n-layer", type=int, default=2)
    parser.add_argument("--n-head", type=int, default=4)
    parser.add_argument("--n-embd", type=int, default=128)
    parser.add_argument("--learning-rate", type=float, default=3e-4)
    parser.add_argument("--warmup", type=int, default=100)
    parser.add_argument("--seed", type=int, default=1337)
    parser.add_argument("--prompt", default=DEFAULT_PROMPT)
    parser.add_argument("--tokens", type=int, default=DEFAULT_TOKENS)
    parser.add_argument("--eval-interval", type=int, default=250)
    parser.add_argument("--eval-batches", type=int, default=32)
    parser.add_argument("--device", choices=("auto", "cpu", "cuda"), default="auto")
    parser.add_argument("--resume", action="store_true")
    parser.add_argument(
        "--export-only",
        action="store_true",
        help="load --checkpoint and write model.txm without training",
    )
    args = parser.parse_args()

    if args.steps < 1:
        raise SystemExit("--steps must be positive")
    if args.batch_size < 1 or args.block_size < 2:
        raise SystemExit("--batch-size and --block-size are too small")
    if args.n_layer < 1 or args.n_head < 1 or args.n_embd < 1:
        raise SystemExit("model dimensions must be positive")
    if args.n_embd % args.n_head != 0:
        raise SystemExit("--n-embd must be divisible by --n-head")
    if args.tokens < 1:
        raise SystemExit("--tokens must be positive")

    if args.export_only:
        checkpoint = load_checkpoint(args.checkpoint, torch.device("cpu"))
        chars = list(checkpoint["chars"])
        stoi = {ch: index for index, ch in enumerate(chars)}
        config = checkpoint["config"]
        model = TinyGPT(**config)
        state = checkpoint.get("best_state") or checkpoint["model"]
        model.load_state_dict(state)
        reference_export(
            model,
            args.prompt,
            args.tokens,
            chars,
            stoi,
            config,
            args.output,
        )
        return 0

    torch.manual_seed(args.seed)
    if torch.cuda.is_available():
        torch.cuda.manual_seed_all(args.seed)

    if args.device == "cuda" and not torch.cuda.is_available():
        raise SystemExit("--device cuda requested, but CUDA is not available")
    device = torch.device(
        "cuda" if args.device == "cuda" or (args.device == "auto" and torch.cuda.is_available()) else "cpu"
    )
    print("device:", device, flush=True)

    text = load_text(args.data)
    chars, stoi = build_vocab(text)
    vocab_size = len(chars)
    print("corpus: %d characters, %d unique" % (len(text), vocab_size), flush=True)

    data = torch.tensor(encode(text, stoi), dtype=torch.long)
    split = int(0.9 * len(data))
    train_data, val_data = data[:split], data[split:]
    val_x, val_y = fixed_validation_batches(val_data, args.block_size, args.eval_batches)
    val_x, val_y = val_x.to(device), val_y.to(device)

    config = {
        "vocab_size": vocab_size,
        "block_size": args.block_size,
        "n_layer": args.n_layer,
        "n_head": args.n_head,
        "n_embd": args.n_embd,
    }
    model = TinyGPT(**config).to(device)
    optimizer = torch.optim.AdamW(
        model.parameters(),
        lr=args.learning_rate,
        betas=(0.9, 0.95),
        weight_decay=0.1,
    )

    start_step = 0
    best_val = float("inf")
    best_state = None
    if args.resume:
        checkpoint = load_checkpoint(args.checkpoint, device)
        model.load_state_dict(checkpoint["model"])
        optimizer.load_state_dict(checkpoint["optimizer"])
        start_step = int(checkpoint["step"]) + 1
        best_val = float(checkpoint["best_val"])
        best_state = checkpoint.get("best_state")
        if list(checkpoint.get("chars", [])) != chars:
            raise SystemExit("checkpoint vocabulary does not match this corpus")
        print("resumed at step %d, best val %.6f" % (start_step, best_val), flush=True)

    prompt_ids = encode(args.prompt, stoi)
    if len(prompt_ids) + args.tokens > args.block_size:
        raise SystemExit(
            "prompt plus generated tokens is %d, above block size %d"
            % (len(prompt_ids) + args.tokens, args.block_size)
        )

    model.train()
    started = time.time()
    for step in range(start_step, args.steps):
        lr = learning_rate(step, args.steps, args.learning_rate, args.warmup)
        for group in optimizer.param_groups:
            group["lr"] = lr

        x, y = get_batch(train_data, args.batch_size, args.block_size, device)
        _, loss = model(x, y)
        optimizer.zero_grad(set_to_none=True)
        loss.backward()
        torch.nn.utils.clip_grad_norm_(model.parameters(), 1.0)
        optimizer.step()

        if step % 50 == 0:
            elapsed = time.time() - started
            print(
                "step %d loss %.4f lr %.2e elapsed %.1fs"
                % (step, float(loss.item()), lr, elapsed),
                flush=True,
            )

        if (step + 1) % args.eval_interval == 0 or step + 1 == args.steps:
            val_loss = estimate_loss(model, val_x, val_y, args.batch_size)
            improved = val_loss < best_val
            if improved:
                best_val = val_loss
                best_state = {
                    name: tensor.detach().cpu().clone()
                    for name, tensor in model.state_dict().items()
                }
            save_checkpoint(
                args.checkpoint,
                model,
                optimizer,
                step,
                best_val,
                best_state,
                config,
                chars,
            )
            sample = sample_text(
                model, prompt_ids, min(60, args.tokens), chars
            )
            print(
                "eval step %d val %.4f best %.4f%s"
                % (step + 1, val_loss, best_val, " *" if improved else ""),
                flush=True,
            )
            print("sample: %s" % sample.replace("\n", "\\n"), flush=True)

    if best_state is None:
        best_state = {
            name: tensor.detach().cpu().clone()
            for name, tensor in model.state_dict().items()
        }
        best_val = estimate_loss(model, val_x, val_y, args.batch_size)
    model.load_state_dict(best_state)
    print("best validation loss: %.6f" % best_val, flush=True)

    reference_export(
        model,
        args.prompt,
        args.tokens,
        chars,
        stoi,
        config,
        args.output,
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
