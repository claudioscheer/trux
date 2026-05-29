package lexer

import "github.com/claudioscheer/trux/internal/token"

type Lexer struct {
	input  string
	offset int
	line   int
	column int
}

func New(input string) *Lexer {
	return &Lexer{
		input:  input,
		line:   1,
		column: 1,
	}
}

func Lex(input string) []token.Token {
	l := New(input)
	tokens := []token.Token{}

	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == token.EOF {
			return tokens
		}
	}
}

func (l *Lexer) NextToken() token.Token {
	l.skipWhitespace()

	pos := l.position()

	if l.atEnd() {
		return token.Token{Type: token.EOF, Pos: pos}
	}

	ch := l.current()

	switch ch {
	case '=':
		l.advance()
		return token.Token{Type: token.Assign, Lexeme: "=", Pos: pos}
	case '+':
		l.advance()
		return token.Token{Type: token.Plus, Lexeme: "+", Pos: pos}
	case '-':
		l.advance()
		return token.Token{Type: token.Minus, Lexeme: "-", Pos: pos}
	case '*':
		l.advance()
		return token.Token{Type: token.Asterisk, Lexeme: "*", Pos: pos}
	case '/':
		l.advance()
		return token.Token{Type: token.Slash, Lexeme: "/", Pos: pos}
	case ',':
		l.advance()
		return token.Token{Type: token.Comma, Lexeme: ",", Pos: pos}
	case '(':
		l.advance()
		return token.Token{Type: token.LParen, Lexeme: "(", Pos: pos}
	case ')':
		l.advance()
		return token.Token{Type: token.RParen, Lexeme: ")", Pos: pos}
	case '{':
		l.advance()
		return token.Token{Type: token.LBrace, Lexeme: "{", Pos: pos}
	case '}':
		l.advance()
		return token.Token{Type: token.RBrace, Lexeme: "}", Pos: pos}
	}

	if isIdentStart(ch) {
		lit := l.readIdentifier()
		return token.Token{Type: token.LookupIdent(lit), Lexeme: lit, Pos: pos}
	}

	if isDigit(ch) {
		lit := l.readInteger()
		return token.Token{Type: token.Int, Lexeme: lit, Pos: pos}
	}

	l.advance()
	return token.Token{Type: token.Illegal, Lexeme: string(ch), Pos: pos}
}

func (l *Lexer) readIdentifier() string {
	start := l.offset

	for !l.atEnd() && isIdentPart(l.current()) {
		l.advance()
	}

	return l.input[start:l.offset]
}

func (l *Lexer) readInteger() string {
	start := l.offset

	for !l.atEnd() && isDigit(l.current()) {
		l.advance()
	}

	return l.input[start:l.offset]
}

func (l *Lexer) skipWhitespace() {
	for !l.atEnd() {
		switch l.current() {
		case ' ', '\t', '\r', '\n':
			l.advance()
		default:
			return
		}
	}
}

func (l *Lexer) advance() byte {
	ch := l.current()
	l.offset++

	if ch == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}

	return ch
}

func (l *Lexer) current() byte {
	return l.input[l.offset]
}

func (l *Lexer) atEnd() bool {
	return l.offset >= len(l.input)
}

func (l *Lexer) position() token.Position {
	return token.Position{
		Offset: l.offset,
		Line:   l.line,
		Column: l.column,
	}
}

func isIdentStart(ch byte) bool {
	return ch == '_' || ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z')
}

func isIdentPart(ch byte) bool {
	return isIdentStart(ch) || isDigit(ch)
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
