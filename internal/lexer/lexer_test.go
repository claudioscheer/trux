package lexer

import (
	"os"
	"testing"

	"github.com/claudioscheer/trux/internal/token"
)

func TestLexesV0Example(t *testing.T) {
	input, err := os.ReadFile("../../examples/v0/hello.tx")
	if err != nil {
		t.Fatal(err)
	}

	want := []expectedToken{
		{token.Package, "package"},
		{token.Ident, "main"},
		{token.Func, "func"},
		{token.Ident, "add"},
		{token.LParen, "("},
		{token.Ident, "a"},
		{token.IntType, "int"},
		{token.Comma, ","},
		{token.Ident, "b"},
		{token.IntType, "int"},
		{token.RParen, ")"},
		{token.IntType, "int"},
		{token.LBrace, "{"},
		{token.Return, "return"},
		{token.Ident, "a"},
		{token.Plus, "+"},
		{token.Ident, "b"},
		{token.RBrace, "}"},
		{token.Func, "func"},
		{token.Ident, "main"},
		{token.LParen, "("},
		{token.RParen, ")"},
		{token.IntType, "int"},
		{token.LBrace, "{"},
		{token.Let, "let"},
		{token.Ident, "x"},
		{token.IntType, "int"},
		{token.Assign, "="},
		{token.Ident, "add"},
		{token.LParen, "("},
		{token.Int, "1"},
		{token.Comma, ","},
		{token.Int, "2"},
		{token.RParen, ")"},
		{token.Ident, "print"},
		{token.LParen, "("},
		{token.Ident, "x"},
		{token.RParen, ")"},
		{token.Return, "return"},
		{token.Int, "0"},
		{token.RBrace, "}"},
		{token.EOF, ""},
	}

	assertTokens(t, Lex(string(input)), want)
}

func TestTracksTokenPositions(t *testing.T) {
	tokens := Lex("package main\n  let x int = 10\n")

	want := []token.Token{
		{Type: token.Package, Lexeme: "package", Pos: token.Position{Offset: 0, Line: 1, Column: 1}},
		{Type: token.Ident, Lexeme: "main", Pos: token.Position{Offset: 8, Line: 1, Column: 9}},
		{Type: token.Let, Lexeme: "let", Pos: token.Position{Offset: 15, Line: 2, Column: 3}},
		{Type: token.Ident, Lexeme: "x", Pos: token.Position{Offset: 19, Line: 2, Column: 7}},
		{Type: token.IntType, Lexeme: "int", Pos: token.Position{Offset: 21, Line: 2, Column: 9}},
		{Type: token.Assign, Lexeme: "=", Pos: token.Position{Offset: 25, Line: 2, Column: 13}},
		{Type: token.Int, Lexeme: "10", Pos: token.Position{Offset: 27, Line: 2, Column: 15}},
		{Type: token.EOF, Pos: token.Position{Offset: 30, Line: 3, Column: 1}},
	}

	if len(tokens) != len(want) {
		t.Fatalf("got %d tokens, want %d", len(tokens), len(want))
	}

	for i := range want {
		if tokens[i] != want[i] {
			t.Fatalf("token %d = %#v, want %#v", i, tokens[i], want[i])
		}
	}
}

func TestPrintIsIdentifier(t *testing.T) {
	assertTokens(t, Lex("print(x)"), []expectedToken{
		{token.Ident, "print"},
		{token.LParen, "("},
		{token.Ident, "x"},
		{token.RParen, ")"},
		{token.EOF, ""},
	})
}

func TestIllegalToken(t *testing.T) {
	tokens := Lex("@")

	want := token.Token{
		Type:   token.Illegal,
		Lexeme: "@",
		Pos:    token.Position{Offset: 0, Line: 1, Column: 1},
	}
	if tokens[0] != want {
		t.Fatalf("first token = %#v, want %#v", tokens[0], want)
	}
}

type expectedToken struct {
	typ    token.Type
	lexeme string
}

func assertTokens(t *testing.T, got []token.Token, want []expectedToken) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("got %d tokens, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i].Type != want[i].typ || got[i].Lexeme != want[i].lexeme {
			t.Fatalf("token %d = (%s, %q), want (%s, %q)", i, got[i].Type, got[i].Lexeme, want[i].typ, want[i].lexeme)
		}
	}
}
