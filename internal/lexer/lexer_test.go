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

func TestLexesV1LiteralsAndTypes(t *testing.T) {
	assertTokens(t, Lex(`let name string = "Kern\n" let ready bool = true false`), []expectedToken{
		{token.Let, "let"},
		{token.Ident, "name"},
		{token.StringType, "string"},
		{token.Assign, "="},
		{token.String, "Kern\n"},
		{token.Let, "let"},
		{token.Ident, "ready"},
		{token.BoolType, "bool"},
		{token.Assign, "="},
		{token.True, "true"},
		{token.False, "false"},
		{token.EOF, ""},
	})
}

func TestLexesV2Tokens(t *testing.T) {
	assertTokens(t, Lex(`if x <= 10 { y = 1.5 } else { while "x" in name != false { y = y + 1.0 } }`), []expectedToken{
		{token.If, "if"},
		{token.Ident, "x"},
		{token.LessEqual, "<="},
		{token.Int, "10"},
		{token.LBrace, "{"},
		{token.Ident, "y"},
		{token.Assign, "="},
		{token.Float, "1.5"},
		{token.RBrace, "}"},
		{token.Else, "else"},
		{token.LBrace, "{"},
		{token.While, "while"},
		{token.String, "x"},
		{token.In, "in"},
		{token.Ident, "name"},
		{token.NotEqual, "!="},
		{token.False, "false"},
		{token.LBrace, "{"},
		{token.Ident, "y"},
		{token.Assign, "="},
		{token.Ident, "y"},
		{token.Plus, "+"},
		{token.Float, "1.0"},
		{token.RBrace, "}"},
		{token.RBrace, "}"},
		{token.EOF, ""},
	})
}

func TestLexesV3CollectionTokens(t *testing.T) {
	assertTokens(t, Lex(`let xs [3]int = [3]int{1, 2, 3} let ys []int = make([]int, len(xs)) append(list[int]{}, xs[0:2][1])`), []expectedToken{
		{token.Let, "let"},
		{token.Ident, "xs"},
		{token.LBracket, "["},
		{token.Int, "3"},
		{token.RBracket, "]"},
		{token.IntType, "int"},
		{token.Assign, "="},
		{token.LBracket, "["},
		{token.Int, "3"},
		{token.RBracket, "]"},
		{token.IntType, "int"},
		{token.LBrace, "{"},
		{token.Int, "1"},
		{token.Comma, ","},
		{token.Int, "2"},
		{token.Comma, ","},
		{token.Int, "3"},
		{token.RBrace, "}"},
		{token.Let, "let"},
		{token.Ident, "ys"},
		{token.LBracket, "["},
		{token.RBracket, "]"},
		{token.IntType, "int"},
		{token.Assign, "="},
		{token.Ident, "make"},
		{token.LParen, "("},
		{token.LBracket, "["},
		{token.RBracket, "]"},
		{token.IntType, "int"},
		{token.Comma, ","},
		{token.Ident, "len"},
		{token.LParen, "("},
		{token.Ident, "xs"},
		{token.RParen, ")"},
		{token.RParen, ")"},
		{token.Ident, "append"},
		{token.LParen, "("},
		{token.List, "list"},
		{token.LBracket, "["},
		{token.IntType, "int"},
		{token.RBracket, "]"},
		{token.LBrace, "{"},
		{token.RBrace, "}"},
		{token.Comma, ","},
		{token.Ident, "xs"},
		{token.LBracket, "["},
		{token.Int, "0"},
		{token.Colon, ":"},
		{token.Int, "2"},
		{token.RBracket, "]"},
		{token.LBracket, "["},
		{token.Int, "1"},
		{token.RBracket, "]"},
		{token.RParen, ")"},
		{token.EOF, ""},
	})
}

func TestLexesStringEscapes(t *testing.T) {
	assertTokens(t, Lex(`"quote: \" slash: \\ tab:\t"`), []expectedToken{
		{token.String, "quote: \" slash: \\ tab:\t"},
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

func TestIllegalStringLiterals(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "single quote", input: `'bad'`, want: "'"},
		{name: "unknown escape", input: `"bad\x"`, want: `error: unknown escape \x`},
		{name: "unterminated", input: `"bad`, want: "error: unterminated string literal"},
		{name: "newline", input: "\"bad\n\"", want: "error: newline in string literal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := Lex(tt.input)
			if tokens[0].Type != token.Illegal || tokens[0].Lexeme != tt.want {
				t.Fatalf("first token = %#v, want illegal %q", tokens[0], tt.want)
			}
		})
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
