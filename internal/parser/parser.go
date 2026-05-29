package parser

import (
	"fmt"
	"strings"

	"github.com/claudioscheer/trux/internal/ast"
	"github.com/claudioscheer/trux/internal/lexer"
	"github.com/claudioscheer/trux/internal/token"
)

type ParseError struct {
	Pos token.Position
	Msg string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error at %d:%d: %s", e.Pos.Line, e.Pos.Column, e.Msg)
}

type Parser struct {
	tokens []token.Token
	pos    int
}

func Parse(input string) (*ast.Program, error) {
	tokens := lexer.Lex(input)
	for _, tok := range tokens {
		if tok.Type == token.Illegal {
			if strings.HasPrefix(tok.Lexeme, "error: ") {
				return nil, &ParseError{
					Pos: tok.Pos,
					Msg: strings.TrimPrefix(tok.Lexeme, "error: "),
				}
			}
			return nil, &ParseError{
				Pos: tok.Pos,
				Msg: fmt.Sprintf("lex error: illegal character %q", tok.Lexeme),
			}
		}
	}

	p := &Parser{tokens: tokens}
	return p.parseProgram()
}

func (p *Parser) parseProgram() (*ast.Program, error) {
	if _, err := p.expect(token.Package); err != nil {
		return nil, err
	}

	packageName, err := p.expect(token.Ident)
	if err != nil {
		return nil, err
	}

	program := &ast.Program{PackageName: packageName.Lexeme}

	for !p.check(token.EOF) {
		fn, err := p.parseFuncDecl()
		if err != nil {
			return nil, err
		}
		program.Functions = append(program.Functions, fn)
	}

	return program, nil
}

func (p *Parser) parseFuncDecl() (*ast.FuncDecl, error) {
	start, err := p.expect(token.Func)
	if err != nil {
		return nil, err
	}

	name, err := p.expect(token.Ident)
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(token.LParen); err != nil {
		return nil, err
	}

	params, err := p.parseParams()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(token.RParen); err != nil {
		return nil, err
	}

	returnType, err := p.parseType()
	if err != nil {
		return nil, err
	}

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	return &ast.FuncDecl{
		Pos:        start.Pos,
		Name:       name.Lexeme,
		Params:     params,
		ReturnType: returnType,
		Body:       body,
	}, nil
}

func (p *Parser) parseParams() ([]ast.Param, error) {
	if p.check(token.RParen) {
		return nil, nil
	}

	params := []ast.Param{}
	for {
		name, err := p.expect(token.Ident)
		if err != nil {
			return nil, err
		}

		typ, err := p.parseType()
		if err != nil {
			return nil, err
		}

		params = append(params, ast.Param{Name: name.Lexeme, Type: typ})

		if !p.match(token.Comma) {
			return params, nil
		}
	}
}

func (p *Parser) parseType() (ast.Type, error) {
	switch {
	case p.match(token.IntType):
		return ast.IntType, nil
	case p.match(token.StringType):
		return ast.StringType, nil
	case p.match(token.BoolType):
		return ast.BoolType, nil
	default:
		tok := p.current()
		return "", p.errorf(tok, "expected type, got %s", describe(tok))
	}
}

func (p *Parser) parseBlock() (ast.Block, error) {
	if _, err := p.expect(token.LBrace); err != nil {
		return ast.Block{}, err
	}

	block := ast.Block{}
	for !p.check(token.RBrace) && !p.check(token.EOF) {
		stmt, err := p.parseStatement()
		if err != nil {
			return ast.Block{}, err
		}
		block.Statements = append(block.Statements, stmt)
	}

	if _, err := p.expect(token.RBrace); err != nil {
		return ast.Block{}, err
	}

	return block, nil
}

func (p *Parser) parseStatement() (ast.Statement, error) {
	switch {
	case p.check(token.Let):
		return p.parseLetStmt()
	case p.check(token.Return):
		return p.parseReturnStmt()
	default:
		return p.parseExprStmt()
	}
}

func (p *Parser) parseLetStmt() (ast.Statement, error) {
	start, err := p.expect(token.Let)
	if err != nil {
		return nil, err
	}

	name, err := p.expect(token.Ident)
	if err != nil {
		return nil, err
	}

	typ, err := p.parseType()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(token.Assign); err != nil {
		return nil, err
	}

	value, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	return &ast.LetStmt{Start: start.Pos, Name: name.Lexeme, Type: typ, Value: value}, nil
}

func (p *Parser) parseReturnStmt() (ast.Statement, error) {
	start, err := p.expect(token.Return)
	if err != nil {
		return nil, err
	}

	value, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	return &ast.ReturnStmt{Start: start.Pos, Value: value}, nil
}

func (p *Parser) parseExprStmt() (ast.Statement, error) {
	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	return &ast.ExprStmt{Expr: expr}, nil
}

func (p *Parser) parseExpression(minPrecedence int) (ast.Expression, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		precedence := p.precedence(p.current().Type)
		if precedence < minPrecedence {
			return left, nil
		}

		operator := p.advance()
		right, err := p.parseExpression(precedence + 1)
		if err != nil {
			return nil, err
		}

		left = &ast.BinaryExpr{
			Start:    left.Pos(),
			Left:     left,
			Operator: operator.Lexeme,
			Right:    right,
		}
	}
}

func (p *Parser) parsePrimary() (ast.Expression, error) {
	switch {
	case p.check(token.Int):
		tok := p.advance()
		return &ast.IntLiteral{Start: tok.Pos, Value: tok.Lexeme}, nil
	case p.check(token.String):
		tok := p.advance()
		return &ast.StringLiteral{Start: tok.Pos, Value: tok.Lexeme}, nil
	case p.check(token.True):
		tok := p.advance()
		return &ast.BoolLiteral{Start: tok.Pos, Value: true}, nil
	case p.check(token.False):
		tok := p.advance()
		return &ast.BoolLiteral{Start: tok.Pos, Value: false}, nil
	case p.check(token.Ident):
		ident := p.advance()
		if p.match(token.LParen) {
			args, err := p.parseArguments()
			if err != nil {
				return nil, err
			}

			if _, err := p.expect(token.RParen); err != nil {
				return nil, err
			}

			return &ast.CallExpr{Start: ident.Pos, Callee: ident.Lexeme, Args: args}, nil
		}

		return &ast.IdentExpr{Start: ident.Pos, Name: ident.Lexeme}, nil
	case p.match(token.LParen):
		expr, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}

		if _, err := p.expect(token.RParen); err != nil {
			return nil, err
		}

		return expr, nil
	default:
		tok := p.current()
		return nil, p.errorf(tok, "expected expression, got %s", describe(tok))
	}
}

func (p *Parser) parseArguments() ([]ast.Expression, error) {
	if p.check(token.RParen) {
		return nil, nil
	}

	args := []ast.Expression{}
	for {
		arg, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)

		if !p.match(token.Comma) {
			return args, nil
		}
	}
}

func (p *Parser) precedence(typ token.Type) int {
	switch typ {
	case token.Plus, token.Minus:
		return 1
	case token.Asterisk, token.Slash:
		return 2
	default:
		return -1
	}
}

func (p *Parser) expect(typ token.Type) (token.Token, error) {
	if p.check(typ) {
		return p.advance(), nil
	}

	tok := p.current()
	return token.Token{}, p.errorf(tok, "expected %s, got %s", describeExpected(typ), describe(tok))
}

func (p *Parser) match(typ token.Type) bool {
	if !p.check(typ) {
		return false
	}

	p.advance()
	return true
}

func (p *Parser) check(typ token.Type) bool {
	return p.current().Type == typ
}

func (p *Parser) advance() token.Token {
	tok := p.current()
	if !p.check(token.EOF) {
		p.pos++
	}
	return tok
}

func (p *Parser) current() token.Token {
	return p.tokens[p.pos]
}

func (p *Parser) errorf(tok token.Token, format string, args ...any) error {
	return &ParseError{
		Pos: tok.Pos,
		Msg: fmt.Sprintf(format, args...),
	}
}

func describeExpected(typ token.Type) string {
	if isLiteralToken(typ) {
		return fmt.Sprintf("%q", string(typ))
	}

	return string(typ)
}

func describe(tok token.Token) string {
	if tok.Type == token.EOF {
		return "end of file"
	}

	if isLiteralToken(tok.Type) || tok.Lexeme != "" {
		return fmt.Sprintf("%q", tok.Lexeme)
	}

	return string(tok.Type)
}

func isLiteralToken(typ token.Type) bool {
	switch typ {
	case token.Assign, token.Plus, token.Minus, token.Asterisk, token.Slash,
		token.Comma, token.LParen, token.RParen, token.LBrace, token.RBrace:
		return true
	default:
		return false
	}
}
