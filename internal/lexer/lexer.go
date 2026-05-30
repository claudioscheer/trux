package lexer

import "github.com/claudioscheer/trux/internal/token"

type Lexer struct {
	file   string
	input  string
	offset int
	line   int
	column int
}

func New(input string) *Lexer {
	return NewFile("", input)
}

func NewFile(file string, input string) *Lexer {
	return &Lexer{
		file:   file,
		input:  input,
		line:   1,
		column: 1,
	}
}

func Lex(input string) []token.Token {
	return LexFile("", input)
}

func LexFile(file string, input string) []token.Token {
	l := NewFile(file, input)
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
	case '"':
		return l.readString(pos)
	case '=':
		l.advance()
		if l.match('=') {
			return token.Token{Type: token.Equal, Lexeme: "==", Pos: pos}
		}
		return token.Token{Type: token.Assign, Lexeme: "=", Pos: pos}
	case '!':
		l.advance()
		if l.match('=') {
			return token.Token{Type: token.NotEqual, Lexeme: "!=", Pos: pos}
		}
		return token.Token{Type: token.Illegal, Lexeme: "!", Pos: pos}
	case '<':
		l.advance()
		if l.match('=') {
			return token.Token{Type: token.LessEqual, Lexeme: "<=", Pos: pos}
		}
		return token.Token{Type: token.Less, Lexeme: "<", Pos: pos}
	case '>':
		l.advance()
		if l.match('=') {
			return token.Token{Type: token.GreaterEqual, Lexeme: ">=", Pos: pos}
		}
		return token.Token{Type: token.Greater, Lexeme: ">", Pos: pos}
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
	case ':':
		l.advance()
		return token.Token{Type: token.Colon, Lexeme: ":", Pos: pos}
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
	case '[':
		l.advance()
		return token.Token{Type: token.LBracket, Lexeme: "[", Pos: pos}
	case ']':
		l.advance()
		return token.Token{Type: token.RBracket, Lexeme: "]", Pos: pos}
	}

	if isIdentStart(ch) {
		lit := l.readIdentifier()
		return token.Token{Type: token.LookupIdent(lit), Lexeme: lit, Pos: pos}
	}

	if isDigit(ch) {
		return l.readNumber(pos)
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

func (l *Lexer) readNumber(pos token.Position) token.Token {
	start := l.offset

	for !l.atEnd() && isDigit(l.current()) {
		l.advance()
	}

	if !l.atEnd() && l.current() == '.' && l.hasNext() && isDigit(l.peek()) {
		l.advance()
		for !l.atEnd() && isDigit(l.current()) {
			l.advance()
		}
		return token.Token{Type: token.Float, Lexeme: l.input[start:l.offset], Pos: pos}
	}

	return token.Token{Type: token.Int, Lexeme: l.input[start:l.offset], Pos: pos}
}

func (l *Lexer) readString(pos token.Position) token.Token {
	l.advance()

	value := []byte{}
	for !l.atEnd() {
		ch := l.current()
		switch ch {
		case '"':
			l.advance()
			return token.Token{Type: token.String, Lexeme: string(value), Pos: pos}
		case '\n', '\r':
			return token.Token{Type: token.Illegal, Lexeme: "error: newline in string literal", Pos: pos}
		case '\\':
			l.advance()
			if l.atEnd() {
				return token.Token{Type: token.Illegal, Lexeme: "error: unterminated string literal", Pos: pos}
			}
			escaped := l.current()
			switch escaped {
			case '"':
				value = append(value, '"')
			case '\\':
				value = append(value, '\\')
			case 'n':
				value = append(value, '\n')
			case 't':
				value = append(value, '\t')
			default:
				return token.Token{Type: token.Illegal, Lexeme: "error: unknown escape \\" + string(escaped), Pos: pos}
			}
			l.advance()
		default:
			value = append(value, ch)
			l.advance()
		}
	}

	return token.Token{Type: token.Illegal, Lexeme: "error: unterminated string literal", Pos: pos}
}

func (l *Lexer) skipWhitespace() {
	for !l.atEnd() {
		switch l.current() {
		case ' ', '\t', '\r', '\n':
			l.advance()
		case '/':
			if !l.hasNext() || l.peek() != '/' {
				return
			}
			for !l.atEnd() && l.current() != '\n' && l.current() != '\r' {
				l.advance()
			}
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

func (l *Lexer) match(ch byte) bool {
	if l.atEnd() || l.current() != ch {
		return false
	}

	l.advance()
	return true
}

func (l *Lexer) current() byte {
	return l.input[l.offset]
}

func (l *Lexer) peek() byte {
	return l.input[l.offset+1]
}

func (l *Lexer) hasNext() bool {
	return l.offset+1 < len(l.input)
}

func (l *Lexer) atEnd() bool {
	return l.offset >= len(l.input)
}

func (l *Lexer) position() token.Position {
	return token.Position{
		File:   l.file,
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
