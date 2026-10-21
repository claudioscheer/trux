"""Pygments lexer for Trux (.tx) source files.

The token lists mirror internal/token/token.go, the builtins handled in
internal/types/types.go, and the standard packages in internal/stdlib.
Keep them in sync when the language grows.
"""

from pygments.lexer import RegexLexer, bygroups, words
from pygments.token import (
    Comment,
    Keyword,
    Name,
    Number,
    Operator,
    Punctuation,
    String,
    Text,
    Whitespace,
)

__all__ = ["TruxLexer"]


class TruxLexer(RegexLexer):
    name = "Trux"
    aliases = ["trux", "tx"]
    filenames = ["*.tx"]
    mimetypes = ["text/x-trux"]

    tokens = {
        "root": [
            (r"\s+", Whitespace),
            (r"//[^\n\r]*", Comment.Single),
            (r'"', String, "string"),
            (r"(package|import)\b", Keyword.Namespace),
            (
                words(
                    (
                        "pub",
                        "kernel",
                        "func",
                        "return",
                        "let",
                        "mut",
                        "if",
                        "else",
                        "for",
                        "break",
                        "continue",
                        "in",
                    ),
                    suffix=r"\b",
                ),
                Keyword,
            ),
            (words(("true", "false"), suffix=r"\b"), Keyword.Constant),
            (words(("int", "float", "string", "bool", "list"), suffix=r"\b"), Keyword.Type),
            (r"(gpu)(\.)(Buffer)\b", bygroups(Name.Namespace, Punctuation, Keyword.Type)),
            (
                r"(io|csv|image|time|http|gpu)(\.)([A-Za-z_][A-Za-z0-9_]*)",
                bygroups(Name.Namespace, Punctuation, Name.Builtin),
            ),
            (
                words(("print", "assert", "len", "append", "clone", "make"), suffix=r"\b"),
                Name.Builtin,
            ),
            (r"(func)(\s+)([A-Za-z_][A-Za-z0-9_]*)", bygroups(Keyword, Whitespace, Name.Function)),
            (r"[A-Za-z_][A-Za-z0-9_]*(?=\s*\()", Name.Function),
            (r"[A-Za-z_][A-Za-z0-9_]*", Name),
            (r"\d+\.\d+", Number.Float),
            (r"\d+", Number.Integer),
            (r"==|!=|<=|>=|&&|\|\||[=!<>+\-*/%]", Operator),
            (r"[(){}\[\],:;.]", Punctuation),
            (r".", Text),
        ],
        "string": [
            (r'\\["\\nt]', String.Escape),
            (r'"', String, "#pop"),
            (r'[^"\\\n]+', String),
            (r"\\.", String.Escape),
        ],
    }
