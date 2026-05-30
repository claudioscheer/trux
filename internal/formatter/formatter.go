package formatter

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/claudioscheer/trux/internal/lexer"
	"github.com/claudioscheer/trux/internal/parser"
	"github.com/claudioscheer/trux/internal/token"
)

const indent = "\t"

func Format(path string, src string) (string, error) {
	if _, err := parser.ParseFile(path, src); err != nil {
		return "", err
	}

	formatted, err := formatSource(path, src)
	if err != nil {
		return "", err
	}

	if _, err := parser.ParseFile(path, formatted); err != nil {
		return "", fmt.Errorf("formatted source is invalid: %w", err)
	}

	return formatted, nil
}

func formatSource(path string, src string) (string, error) {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")

	lines := strings.Split(src, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	indentLevel := 0
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		code, comment := splitLineComment(line)
		code = strings.TrimSpace(code)
		comment = strings.TrimSpace(comment)

		if code == "" && comment == "" {
			out = append(out, "")
			continue
		}

		leadingClosers := countLeading(code, '}')
		if leadingClosers > indentLevel {
			indentLevel = 0
		} else {
			indentLevel -= leadingClosers
		}

		formattedCode := ""
		if code != "" {
			var err error
			formattedCode, err = formatCodeLine(path, code)
			if err != nil {
				return "", err
			}
		}

		var builder strings.Builder
		builder.WriteString(strings.Repeat(indent, indentLevel))
		builder.WriteString(formattedCode)
		if formattedCode != "" && comment != "" {
			builder.WriteByte(' ')
		}
		builder.WriteString(comment)
		out = append(out, builder.String())

		indentLevel += braceDelta(code, leadingClosers)
		if indentLevel < 0 {
			indentLevel = 0
		}
	}

	return strings.Join(out, "\n") + "\n", nil
}

func splitLineComment(line string) (string, string) {
	inString := false
	escaped := false

	for i := 0; i < len(line)-1; i++ {
		ch := line[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch ch {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}

		if ch == '"' {
			inString = true
			continue
		}
		if ch == '/' && line[i+1] == '/' {
			return line[:i], line[i:]
		}
	}

	return line, ""
}

func formatCodeLine(path string, line string) (string, error) {
	tokens := lexer.LexFile(path, line)
	var builder strings.Builder
	var prevprev token.Token
	var prev token.Token
	hasPrev := false

	for _, tok := range tokens {
		if tok.Type == token.EOF {
			break
		}
		if tok.Type == token.Illegal {
			return "", fmt.Errorf("%d:%d: illegal token %q", tok.Pos.Line, tok.Pos.Column, tok.Lexeme)
		}

		if hasPrev && needsSpace(prevprev, prev, tok) {
			builder.WriteByte(' ')
		}
		builder.WriteString(formatLexeme(tok))

		prevprev = prev
		prev = tok
		hasPrev = true
	}

	return builder.String(), nil
}

func needsSpace(prevprev token.Token, prev token.Token, curr token.Token) bool {
	if noSpaceBefore(curr.Type) || noSpaceAfter(prev.Type) {
		return false
	}

	if curr.Type == token.LParen && prev.Type == token.Ident {
		return false
	}

	if curr.Type == token.LBracket {
		switch prev.Type {
		case token.List, token.RBracket:
			return false
		case token.Ident:
			return prevprev.Type == token.Let || prevprev.Type == token.LParen || prevprev.Type == token.Comma
		default:
			return true
		}
	}

	if isTypeToken(curr.Type) && prev.Type == token.RBracket {
		return false
	}

	if curr.Type == token.LBrace {
		if prev.Type == token.RBracket || (isTypeToken(prev.Type) && prevprev.Type == token.RBracket) {
			return false
		}
		return true
	}

	if isOperator(curr.Type) || isOperator(prev.Type) {
		return true
	}

	if prev.Type == token.Comma {
		return true
	}

	return true
}

func noSpaceBefore(typ token.Type) bool {
	switch typ {
	case token.RParen, token.RBracket, token.Comma, token.Colon:
		return true
	default:
		return false
	}
}

func noSpaceAfter(typ token.Type) bool {
	switch typ {
	case token.LParen, token.LBracket, token.Colon:
		return true
	default:
		return false
	}
}

func isOperator(typ token.Type) bool {
	switch typ {
	case token.Assign, token.Equal, token.NotEqual, token.Less, token.LessEqual, token.Greater, token.GreaterEqual,
		token.Plus, token.Minus, token.Asterisk, token.Slash, token.In:
		return true
	default:
		return false
	}
}

func isTypeToken(typ token.Type) bool {
	switch typ {
	case token.IntType, token.FloatType, token.StringType, token.BoolType:
		return true
	default:
		return false
	}
}

func formatLexeme(tok token.Token) string {
	if tok.Type == token.String {
		return quoteString(tok.Lexeme)
	}
	return tok.Lexeme
}

func quoteString(value string) string {
	var builder strings.Builder
	builder.WriteByte('"')
	for _, r := range value {
		switch r {
		case '"':
			builder.WriteString(`\"`)
		case '\\':
			builder.WriteString(`\\`)
		case '\n':
			builder.WriteString(`\n`)
		case '\t':
			builder.WriteString(`\t`)
		default:
			if strconv.IsPrint(r) {
				builder.WriteRune(r)
			} else {
				builder.WriteString(strconv.QuoteRune(r)[1 : len(strconv.QuoteRune(r))-1])
			}
		}
	}
	builder.WriteByte('"')
	return builder.String()
}

func countLeading(text string, ch rune) int {
	count := 0
	for _, r := range text {
		if r != ch {
			return count
		}
		count++
	}
	return count
}

func braceDelta(code string, leadingClosers int) int {
	open := 0
	close := 0
	inString := false
	escaped := false

	for _, r := range code {
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch r {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}

		switch r {
		case '"':
			inString = true
		case '{':
			open++
		case '}':
			close++
		}
	}

	close -= leadingClosers
	if close < 0 {
		close = 0
	}

	return open - close
}
