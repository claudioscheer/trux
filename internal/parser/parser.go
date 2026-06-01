package parser

import (
	"fmt"
	"strconv"
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
	return ParseFile("", input)
}

func ParseFile(path string, input string) (*ast.Program, error) {
	tokens := lexer.LexFile(path, input)
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

	for p.check(token.Import) {
		importDecl, err := p.parseImportDecl()
		if err != nil {
			return nil, err
		}
		program.Imports = append(program.Imports, importDecl)
	}

	for !p.check(token.EOF) {
		if p.check(token.Import) {
			return nil, p.errorf(p.current(), "imports must appear after package and before functions")
		}
		fn, err := p.parseFuncDecl()
		if err != nil {
			return nil, err
		}
		program.Functions = append(program.Functions, fn)
	}

	return program, nil
}

func (p *Parser) parseImportDecl() (*ast.ImportDecl, error) {
	start, err := p.expect(token.Import)
	if err != nil {
		return nil, err
	}

	path, err := p.expect(token.String)
	if err != nil {
		return nil, err
	}

	return &ast.ImportDecl{Pos: start.Pos, Path: path.Lexeme}, nil
}

func (p *Parser) parseFuncDecl() (*ast.FuncDecl, error) {
	public := false
	start := p.current()
	if p.match(token.Pub) {
		public = true
		if _, err := p.expect(token.Func); err != nil {
			return nil, err
		}
	} else {
		var err error
		start, err = p.expect(token.Func)
		if err != nil {
			return nil, err
		}
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
		NamePos:    name.Pos,
		Name:       name.Lexeme,
		Public:     public,
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

		params = append(params, ast.Param{Pos: name.Pos, Name: name.Lexeme, Type: typ})

		if !p.match(token.Comma) {
			return params, nil
		}
	}
}

func (p *Parser) parseType() (ast.Type, error) {
	switch {
	case p.match(token.LBracket):
		if p.match(token.RBracket) {
			elem, err := p.parseType()
			if err != nil {
				return nil, err
			}
			return &ast.SliceType{Elem: elem}, nil
		}

		lengthTok, err := p.expect(token.Int)
		if err != nil {
			return nil, err
		}
		length, err := strconv.Atoi(lengthTok.Lexeme)
		if err != nil || length <= 0 {
			return nil, p.errorf(lengthTok, "array length must be a positive integer literal")
		}
		if _, err := p.expect(token.RBracket); err != nil {
			return nil, err
		}
		elem, err := p.parseType()
		if err != nil {
			return nil, err
		}
		return &ast.ArrayType{Length: length, Elem: elem}, nil
	case p.match(token.List):
		if _, err := p.expect(token.LBracket); err != nil {
			return nil, err
		}
		elem, err := p.parseType()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(token.RBracket); err != nil {
			return nil, err
		}
		return &ast.ListType{Elem: elem}, nil
	case p.match(token.IntType):
		return ast.IntType, nil
	case p.match(token.FloatType):
		return ast.FloatType, nil
	case p.match(token.StringType):
		return ast.StringType, nil
	case p.match(token.BoolType):
		return ast.BoolType, nil
	default:
		tok := p.current()
		return nil, p.errorf(tok, "expected type, got %s", describe(tok))
	}
}

func (p *Parser) parseBlock() (ast.Block, error) {
	start, err := p.expect(token.LBrace)
	if err != nil {
		return ast.Block{}, err
	}

	block := ast.Block{Start: start.Pos}
	for !p.check(token.RBrace) && !p.check(token.EOF) {
		stmt, err := p.parseStatement()
		if err != nil {
			return ast.Block{}, err
		}
		block.Statements = append(block.Statements, stmt)
	}

	end, err := p.expect(token.RBrace)
	if err != nil {
		return ast.Block{}, err
	}
	block.End = end.Pos

	return block, nil
}

func (p *Parser) parseStatement() (ast.Statement, error) {
	switch {
	case p.check(token.Let):
		return p.parseLetStmt()
	case p.check(token.Return):
		return p.parseReturnStmt()
	case p.check(token.If):
		return p.parseIfStmt()
	case p.check(token.For):
		return p.parseForStmt()
	default:
		return p.parseAssignOrExprStmt()
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

	return &ast.LetStmt{Start: start.Pos, NamePos: name.Pos, Name: name.Lexeme, Type: typ, Value: value}, nil
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

func (p *Parser) parseIfStmt() (ast.Statement, error) {
	start, err := p.expect(token.If)
	if err != nil {
		return nil, err
	}

	condition, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	thenBlock, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	var elseBlock *ast.Block
	if p.match(token.Else) {
		if p.check(token.If) {
			elseIf, err := p.parseIfStmt()
			if err != nil {
				return nil, err
			}
			elseBlock = &ast.Block{
				Start:      elseIf.Pos(),
				End:        statementEnd(elseIf),
				Statements: []ast.Statement{elseIf},
			}
		} else {
			block, err := p.parseBlock()
			if err != nil {
				return nil, err
			}
			elseBlock = &block
		}
	}
	if p.check(token.If) && p.current().Pos.Line == ifEnd(thenBlock, elseBlock).Line {
		return nil, p.errorf(p.current(), `unexpected "if" after block on the same line; use "else if" for chained conditionals or put the next if on a new line`)
	}

	return &ast.IfStmt{Start: start.Pos, Condition: condition, Then: thenBlock, Else: elseBlock}, nil
}

func ifEnd(thenBlock ast.Block, elseBlock *ast.Block) token.Position {
	if elseBlock != nil {
		return elseBlock.End
	}
	return thenBlock.End
}

func statementEnd(stmt ast.Statement) token.Position {
	switch stmt := stmt.(type) {
	case *ast.IfStmt:
		return ifEnd(stmt.Then, stmt.Else)
	case *ast.ForStmt:
		return stmt.Body.End
	default:
		return stmt.Pos()
	}
}

func (p *Parser) parseForStmt() (ast.Statement, error) {
	start, err := p.expect(token.For)
	if err != nil {
		return nil, err
	}

	var init ast.Statement
	var condition ast.Expression
	var post ast.Statement

	if !p.check(token.LBrace) {
		first, err := p.parseForInitStmt()
		if err != nil {
			return nil, err
		}

		if p.check(token.LBrace) {
			exprStmt, ok := first.(*ast.ExprStmt)
			if !ok {
				return nil, p.errorf(p.current(), `expected %q after for init statement`, string(token.Semicolon))
			}
			condition = exprStmt.Expr
		} else {
			init = first
			if _, err := p.expect(token.Semicolon); err != nil {
				return nil, err
			}
			if !p.check(token.Semicolon) {
				condition, err = p.parseExpression(0)
				if err != nil {
					return nil, err
				}
			}
			if _, err := p.expect(token.Semicolon); err != nil {
				return nil, err
			}
			if !p.check(token.LBrace) {
				post, err = p.parseForPostStmt()
				if err != nil {
					return nil, err
				}
			}
		}
	}

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	if p.check(token.If) && p.current().Pos.Line == body.End.Line {
		return nil, p.errorf(p.current(), `unexpected "if" after block on the same line; put the next if on a new line`)
	}

	return &ast.ForStmt{Start: start.Pos, Init: init, Condition: condition, Post: post, Body: body}, nil
}

func (p *Parser) parseForInitStmt() (ast.Statement, error) {
	if p.check(token.Semicolon) {
		return nil, nil
	}
	if p.check(token.Let) {
		return p.parseLetStmt()
	}
	return p.parseAssignOrExprStmt()
}

func (p *Parser) parseForPostStmt() (ast.Statement, error) {
	if p.check(token.Let) {
		return nil, p.errorf(p.current(), "for post statement cannot be let")
	}
	return p.parseAssignOrExprStmt()
}

func (p *Parser) parseAssignOrExprStmt() (ast.Statement, error) {
	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	if p.match(token.Assign) {
		value, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}

		switch target := expr.(type) {
		case *ast.IdentExpr:
			return &ast.AssignStmt{Start: target.Start, Name: target.Name, Value: value}, nil
		case *ast.IndexExpr:
			return &ast.IndexAssignStmt{Start: target.Start, Target: target, Value: value}, nil
		default:
			return nil, p.errorf(token.Token{Pos: expr.Pos()}, "invalid assignment target")
		}
	}

	return &ast.ExprStmt{Expr: expr}, nil
}

func (p *Parser) parseExpression(minPrecedence int) (ast.Expression, error) {
	left, err := p.parsePostfix()
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

func (p *Parser) parsePostfix() (ast.Expression, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for p.match(token.LBracket) {
		if p.match(token.Colon) {
			var end ast.Expression
			if !p.check(token.RBracket) {
				end, err = p.parseExpression(0)
				if err != nil {
					return nil, err
				}
			}
			if _, err := p.expect(token.RBracket); err != nil {
				return nil, err
			}
			expr = &ast.SliceExpr{Start: expr.Pos(), Collection: expr, EndIndex: end}
			continue
		}

		index, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		if p.match(token.Colon) {
			var end ast.Expression
			if !p.check(token.RBracket) {
				end, err = p.parseExpression(0)
				if err != nil {
					return nil, err
				}
			}
			if _, err := p.expect(token.RBracket); err != nil {
				return nil, err
			}
			expr = &ast.SliceExpr{Start: expr.Pos(), Collection: expr, StartIndex: index, EndIndex: end}
			continue
		}

		if _, err := p.expect(token.RBracket); err != nil {
			return nil, err
		}
		expr = &ast.IndexExpr{Start: expr.Pos(), Collection: expr, Index: index}
	}

	return expr, nil
}

func (p *Parser) parsePrimary() (ast.Expression, error) {
	switch {
	case p.check(token.LBracket), p.check(token.List):
		return p.parseCollectionLiteral()
	case p.check(token.Int):
		tok := p.advance()
		return &ast.IntLiteral{Start: tok.Pos, Value: tok.Lexeme}, nil
	case p.check(token.Float):
		tok := p.advance()
		return &ast.FloatLiteral{Start: tok.Pos, Value: tok.Lexeme}, nil
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
		packageName := ""
		callee := ident.Lexeme
		if p.match(token.Dot) {
			member, err := p.expect(token.Ident)
			if err != nil {
				return nil, err
			}
			packageName = ident.Lexeme
			callee = member.Lexeme
		}
		if p.match(token.LParen) {
			if packageName == "" && callee == "make" {
				return p.parseMakeExpr(ident)
			}

			args, err := p.parseArguments()
			if err != nil {
				return nil, err
			}

			if _, err := p.expect(token.RParen); err != nil {
				return nil, err
			}

			return &ast.CallExpr{Start: ident.Pos, Package: packageName, Callee: callee, Args: args}, nil
		}
		if packageName != "" {
			return nil, p.errorf(p.current(), "expected %q, got %s", string(token.LParen), describe(p.current()))
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

func (p *Parser) parseCollectionLiteral() (ast.Expression, error) {
	start := p.current().Pos
	typ, err := p.parseType()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(token.LBrace); err != nil {
		return nil, err
	}

	elements, err := p.parseExpressionList(token.RBrace)
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(token.RBrace); err != nil {
		return nil, err
	}

	switch typ.(type) {
	case *ast.ArrayType:
		return &ast.ArrayLiteral{Start: start, Type: typ, Elements: elements}, nil
	case *ast.ListType:
		return &ast.ListLiteral{Start: start, Type: typ, Elements: elements}, nil
	default:
		return nil, p.errorf(token.Token{Pos: start}, "only array and list literals are supported")
	}
}

func (p *Parser) parseMakeExpr(ident token.Token) (ast.Expression, error) {
	typ, err := p.parseType()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(token.Comma); err != nil {
		return nil, err
	}
	length, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(token.RParen); err != nil {
		return nil, err
	}

	return &ast.MakeExpr{Start: ident.Pos, Type: typ, Len: length}, nil
}

func (p *Parser) parseArguments() ([]ast.Expression, error) {
	return p.parseExpressionList(token.RParen)
}

func (p *Parser) parseExpressionList(end token.Type) ([]ast.Expression, error) {
	if p.check(end) {
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
		if p.check(end) {
			return args, nil
		}
	}
}

func (p *Parser) precedence(typ token.Type) int {
	switch typ {
	case token.Equal, token.NotEqual, token.Less, token.LessEqual, token.Greater, token.GreaterEqual, token.In:
		return 0
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

func (p *Parser) peek() token.Token {
	if p.pos+1 >= len(p.tokens) {
		return p.current()
	}

	return p.tokens[p.pos+1]
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
	case token.Assign, token.Equal, token.NotEqual, token.Less, token.LessEqual, token.Greater, token.GreaterEqual,
		token.Plus, token.Minus, token.Asterisk, token.Slash,
		token.Comma, token.Colon, token.Dot, token.Semicolon, token.LParen, token.RParen, token.LBrace, token.RBrace, token.LBracket, token.RBracket:
		return true
	default:
		return false
	}
}
