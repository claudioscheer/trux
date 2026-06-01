package formatter

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/claudioscheer/trux/internal/lexer"
	"github.com/claudioscheer/trux/internal/parser"
	"github.com/claudioscheer/trux/internal/token"
)

const indent = "  "

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

	out = normalizePackageImportSpacing(out)
	return strings.Join(out, "\n") + "\n", nil
}

func normalizePackageImportSpacing(lines []string) []string {
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		out = append(out, line)

		switch {
		case isPackageLine(line):
			next := nextNonBlankLine(lines, i+1)
			if next < 0 {
				i = len(lines)
				continue
			}
			out = append(out, "")
			i = next - 1
		case isImportLine(line):
			next := nextNonBlankLine(lines, i+1)
			if next < 0 {
				i = len(lines)
				continue
			}
			if isImportLine(lines[next]) {
				i = next - 1
				continue
			}
			out = append(out, "")
			i = next - 1
		}
	}
	return out
}

func nextNonBlankLine(lines []string, start int) int {
	for i := start; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			return i
		}
	}
	return -1
}

func isPackageLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "package ")
}

func isImportLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "import ")
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

	for i, tok := range tokens {
		if tok.Type == token.EOF {
			break
		}
		if tok.Type == token.Illegal {
			return "", fmt.Errorf("%d:%d: illegal token %q", tok.Pos.Line, tok.Pos.Column, tok.Lexeme)
		}

		if hasPrev && needsSpace(tokens, i, prevprev, prev, tok) {
			builder.WriteByte(' ')
		}
		builder.WriteString(formatLexeme(tok))

		prevprev = prev
		prev = tok
		hasPrev = true
	}

	return builder.String(), nil
}

func needsSpace(tokens []token.Token, index int, prevprev token.Token, prev token.Token, curr token.Token) bool {
	if noSpaceBefore(curr.Type) || noSpaceAfter(prev.Type) {
		return false
	}

	if curr.Type == token.LParen && prev.Type == token.Ident {
		return false
	}

	if curr.Type == token.RBrace && prev.Type == token.LBrace {
		return false
	}

	if curr.Type == token.RBrace && isCollectionLiteralCloseBrace(tokens, index) {
		return false
	}

	if prev.Type == token.LBrace && isCollectionLiteralOpenBrace(tokens, index-1) {
		return false
	}

	if curr.Type == token.LBracket {
		switch prev.Type {
		case token.List, token.RBracket:
			return false
		case token.Ident:
			return prevprev.Type == token.Let || isFunctionParamType(tokens, index)
		case token.String:
			return false
		case token.RParen:
			return isFunctionReturnType(tokens, index)
		default:
			return true
		}
	}

	if isTypeToken(curr.Type) && prev.Type == token.RBracket {
		return false
	}

	if curr.Type == token.LBrace {
		if isFunctionBodyBrace(tokens, index) {
			return true
		}
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

func isCollectionLiteralCloseBrace(tokens []token.Token, index int) bool {
	lbrace := innermostOpenBrace(tokens, index)
	if lbrace < 0 {
		return false
	}
	return isCollectionLiteralOpenBrace(tokens, lbrace)
}

func isFunctionParamType(tokens []token.Token, index int) bool {
	lparen := innermostOpenParen(tokens, index)
	return isFunctionParamsOpenParen(tokens, lparen)
}

func isFunctionReturnType(tokens []token.Token, index int) bool {
	lparen := matchingOpenParen(tokens, index-1)
	return isFunctionParamsOpenParen(tokens, lparen)
}

func isFunctionBodyBrace(tokens []token.Token, index int) bool {
	for i := 0; i+2 < index; i++ {
		if tokens[i].Type != token.Func || tokens[i+1].Type != token.Ident || tokens[i+2].Type != token.LParen {
			continue
		}

		rparen := matchingCloseParen(tokens, i+2, index)
		if rparen < 0 {
			return false
		}
		for j := rparen + 1; j < index; j++ {
			if tokens[j].Type == token.LBrace {
				return false
			}
		}
		return true
	}
	return false
}

func innermostOpenParen(tokens []token.Token, before int) int {
	depth := 0
	for i := before - 1; i >= 0; i-- {
		switch tokens[i].Type {
		case token.RParen:
			depth++
		case token.LParen:
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

func innermostOpenBrace(tokens []token.Token, before int) int {
	depth := 0
	for i := before - 1; i >= 0; i-- {
		switch tokens[i].Type {
		case token.RBrace:
			depth++
		case token.LBrace:
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

func isCollectionLiteralOpenBrace(tokens []token.Token, lbrace int) bool {
	if lbrace < 1 {
		return false
	}
	switch prev := tokens[lbrace-1]; prev.Type {
	case token.RBracket:
		return true
	case token.IntType, token.FloatType, token.StringType, token.BoolType:
		return lbrace >= 2 && tokens[lbrace-2].Type == token.RBracket
	default:
		return false
	}
}

func matchingOpenParen(tokens []token.Token, rparen int) int {
	if rparen < 0 || rparen >= len(tokens) || tokens[rparen].Type != token.RParen {
		return -1
	}

	depth := 0
	for i := rparen; i >= 0; i-- {
		switch tokens[i].Type {
		case token.RParen:
			depth++
		case token.LParen:
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func matchingCloseParen(tokens []token.Token, lparen int, before int) int {
	if lparen < 0 || lparen >= len(tokens) || tokens[lparen].Type != token.LParen {
		return -1
	}

	depth := 0
	for i := lparen; i < before; i++ {
		switch tokens[i].Type {
		case token.LParen:
			depth++
		case token.RParen:
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func isFunctionParamsOpenParen(tokens []token.Token, lparen int) bool {
	return lparen >= 2 && tokens[lparen-1].Type == token.Ident && tokens[lparen-2].Type == token.Func
}

func noSpaceBefore(typ token.Type) bool {
	switch typ {
	case token.RParen, token.RBracket, token.Comma, token.Colon, token.Dot, token.Semicolon:
		return true
	default:
		return false
	}
}

func noSpaceAfter(typ token.Type) bool {
	switch typ {
	case token.LParen, token.LBracket, token.Colon, token.Dot:
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
