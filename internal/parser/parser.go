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
	tokens            []token.Token
	pos               int
	inConditionHeader bool
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
		if p.check(token.TypeKw) || (p.check(token.Pub) && p.peek(1).Type == token.TypeKw) {
			st, err := p.parseStructDecl()
			if err != nil {
				return nil, err
			}
			program.Structs = append(program.Structs, st)
			continue
		}
		fn, err := p.parseFuncDecl()
		if err != nil {
			return nil, err
		}
		program.Functions = append(program.Functions, fn)
	}

	return program, nil
}

func (p *Parser) parseStructDecl() (*ast.StructDecl, error) {
	public := false
	start := p.current()
	if p.match(token.Pub) {
		public = true
		if _, err := p.expect(token.TypeKw); err != nil {
			return nil, err
		}
	} else {
		var err error
		start, err = p.expect(token.TypeKw)
		if err != nil {
			return nil, err
		}
	}

	name, err := p.expect(token.Ident)
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(token.Struct); err != nil {
		return nil, err
	}

	if _, err := p.expect(token.LBrace); err != nil {
		return nil, err
	}

	var fields []ast.StructField
	for !p.check(token.RBrace) && !p.check(token.EOF) {
		fieldName, err := p.expect(token.Ident)
		if err != nil {
			return nil, err
		}
		fieldType, err := p.parseType()
		if err != nil {
			return nil, err
		}
		prevLine := p.tokens[p.pos-1].Pos.Line
		fields = append(fields, ast.StructField{
			Pos:  fieldName.Pos,
			Name: fieldName.Lexeme,
			Type: fieldType,
		})
		if p.match(token.Comma) || p.match(token.Semicolon) {
			continue
		}
		if !p.check(token.RBrace) && p.current().Pos.Line == prevLine {
			return nil, p.errorf(p.current(), "expected %q or newline after struct field, got %s", ",", describe(p.current()))
		}
	}

	if _, err := p.expect(token.RBrace); err != nil {
		return nil, err
	}

	return &ast.StructDecl{
		Pos:     start.Pos,
		NamePos: name.Pos,
		Name:    name.Lexeme,
		Public:  public,
		Fields:  fields,
	}, nil
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
	kernel := false
	start := p.current()
	if p.match(token.Kernel) {
		kernel = true
		if _, err := p.expect(token.Func); err != nil {
			return nil, err
		}
	} else if p.match(token.Pub) {
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

	var returnType ast.Type
	if !kernel && !p.check(token.LBrace) {
		returnType, err = p.parseType()
		if err != nil {
			return nil, err
		}
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
		Kernel:     kernel,
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
		mutable := p.match(token.Mut)
		name, err := p.expect(token.Ident)
		if err != nil {
			return nil, err
		}

		typ, err := p.parseType()
		if err != nil {
			return nil, err
		}

		params = append(params, ast.Param{Pos: name.Pos, Name: name.Lexeme, Type: typ, Mutable: mutable})

		if !p.match(token.Comma) {
			return params, nil
		}
		if p.check(token.RParen) {
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
	case p.match(token.Asterisk):
		elem, err := p.parseType()
		if err != nil {
			return nil, err
		}
		return &ast.PointerType{Elem: elem}, nil
	case p.match(token.Map):
		if _, err := p.expect(token.LBracket); err != nil {
			return nil, err
		}
		key, err := p.parseType()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(token.RBracket); err != nil {
			return nil, err
		}
		val, err := p.parseType()
		if err != nil {
			return nil, err
		}
		return &ast.MapType{Key: key, Value: val}, nil
	case p.check(token.Ident) && p.current().Lexeme == "gpu":
		p.advance()
		if _, err := p.expect(token.Dot); err != nil {
			return nil, err
		}
		name, err := p.expect(token.Ident)
		if err != nil {
			return nil, err
		}
		if name.Lexeme != "Buffer" {
			return nil, p.errorf(name, "expected gpu.Buffer type, got gpu.%s", name.Lexeme)
		}
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
		return &ast.GPUBufferType{Elem: elem}, nil
	case p.check(token.Ident):
		first := p.advance()
		if p.match(token.Dot) {
			second, err := p.expect(token.Ident)
			if err != nil {
				return nil, err
			}
			return &ast.StructType{Package: first.Lexeme, Name: second.Lexeme}, nil
		}
		return &ast.StructType{Name: first.Lexeme}, nil
	case p.match(token.IntType):
		return ast.IntType, nil
	case p.match(token.FloatType):
		return ast.FloatType, nil
	case p.match(token.StringType):
		return ast.StringType, nil
	case p.match(token.BoolType):
		return ast.BoolType, nil
	case p.match(token.ErrorType):
		return ast.ErrorType, nil
	case p.match(token.LParen):
		var types []ast.Type
		for !p.check(token.RParen) && !p.check(token.EOF) {
			elem, err := p.parseType()
			if err != nil {
				return nil, err
			}
			types = append(types, elem)
			if !p.match(token.Comma) {
				break
			}
		}
		if _, err := p.expect(token.RParen); err != nil {
			return nil, err
		}
		if len(types) == 1 {
			return types[0], nil
		}
		return &ast.TupleType{Types: types}, nil
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
	case p.check(token.Break):
		return p.parseBreakStmt()
	case p.check(token.Continue):
		return p.parseContinueStmt()
	default:
		return p.parseAssignOrExprStmt()
	}
}

func (p *Parser) parseBreakStmt() (ast.Statement, error) {
	start, err := p.expect(token.Break)
	if err != nil {
		return nil, err
	}
	return &ast.BreakStmt{Start: start.Pos}, nil
}

func (p *Parser) parseContinueStmt() (ast.Statement, error) {
	start, err := p.expect(token.Continue)
	if err != nil {
		return nil, err
	}
	return &ast.ContinueStmt{Start: start.Pos}, nil
}

func (p *Parser) parseLetStmt() (ast.Statement, error) {
	start, err := p.expect(token.Let)
	if err != nil {
		return nil, err
	}

	var bindings []ast.LetBinding
	for {
		name, err := p.expect(token.Ident)
		if err != nil {
			return nil, err
		}

		var typ ast.Type
		if name.Lexeme == "_" && (p.check(token.Comma) || p.check(token.Assign)) {
			typ = nil
		} else {
			typ, err = p.parseType()
			if err != nil {
				return nil, err
			}
		}

		bindings = append(bindings, ast.LetBinding{
			Pos:  name.Pos,
			Name: name.Lexeme,
			Type: typ,
		})

		if !p.match(token.Comma) {
			break
		}
	}

	if _, err := p.expect(token.Assign); err != nil {
		return nil, err
	}

	value, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	first := bindings[0]
	return &ast.LetStmt{
		Start:    start.Pos,
		NamePos:  first.Pos,
		Name:     first.Name,
		Type:     first.Type,
		Bindings: bindings,
		Value:    value,
	}, nil
}

func (p *Parser) parseReturnStmt() (ast.Statement, error) {
	start, err := p.expect(token.Return)
	if err != nil {
		return nil, err
	}

	var values []ast.Expression
	if !p.check(token.RBrace) && !p.check(token.EOF) {
		for {
			val, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			values = append(values, val)
			if !p.match(token.Comma) {
				break
			}
		}
	}

	var first ast.Expression
	if len(values) > 0 {
		first = values[0]
	}

	return &ast.ReturnStmt{Start: start.Pos, Value: first, Values: values}, nil
}

func (p *Parser) parseIfStmt() (ast.Statement, error) {
	start, err := p.expect(token.If)
	if err != nil {
		return nil, err
	}

	prevCond := p.inConditionHeader
	p.inConditionHeader = true
	condition, err := p.parseExpression(0)
	p.inConditionHeader = prevCond
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

	prevCond := p.inConditionHeader
	p.inConditionHeader = true
	switch {
	case p.check(token.LBrace):
	case p.forHeaderHasSemicolon():
		init, err = p.parseForInitStmt()
		if err != nil {
			p.inConditionHeader = prevCond
			return nil, err
		}
		if _, err = p.expect(token.Semicolon); err != nil {
			p.inConditionHeader = prevCond
			return nil, err
		}
		if !p.check(token.Semicolon) {
			condition, err = p.parseExpression(0)
			if err != nil {
				p.inConditionHeader = prevCond
				return nil, err
			}
		}
		if _, err = p.expect(token.Semicolon); err != nil {
			p.inConditionHeader = prevCond
			return nil, err
		}
		if !p.check(token.LBrace) {
			post, err = p.parseForPostStmt()
			if err != nil {
				p.inConditionHeader = prevCond
				return nil, err
			}
		}
	default:
		if p.check(token.Let) {
			if _, err = p.parseLetStmt(); err != nil {
				p.inConditionHeader = prevCond
				return nil, err
			}
			p.inConditionHeader = prevCond
			return nil, p.errorf(p.current(), `expected %q after for init statement`, string(token.Semicolon))
		}
		condition, err = p.parseExpression(0)
		if err != nil {
			p.inConditionHeader = prevCond
			return nil, err
		}
	}
	p.inConditionHeader = prevCond

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	if p.check(token.If) && p.current().Pos.Line == body.End.Line {
		return nil, p.errorf(p.current(), `unexpected "if" after block on the same line; put the next if on a new line`)
	}

	return &ast.ForStmt{Start: start.Pos, Init: init, Condition: condition, Post: post, Body: body}, nil
}

func (p *Parser) forHeaderHasSemicolon() bool {
	depth := 0
	for i := p.pos; i < len(p.tokens); i++ {
		switch p.tokens[i].Type {
		case token.LParen, token.LBracket, token.LBrace:
			if p.tokens[i].Type == token.LBrace && depth == 0 && !p.isStructLiteralBraceAt(i) {
				return false
			}
			depth++
		case token.RParen, token.RBracket, token.RBrace:
			if depth > 0 {
				depth--
			}
		case token.Semicolon:
			if depth == 0 {
				return true
			}
		case token.EOF:
			return false
		}
	}
	return false
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

	if p.check(token.Comma) {
		firstIdent, ok := expr.(*ast.IdentExpr)
		if !ok {
			return nil, p.errorf(token.Token{Pos: expr.Pos()}, "invalid multi-assignment target")
		}
		targets := []ast.AssignTarget{{Pos: firstIdent.Start, Name: firstIdent.Name}}
		for p.match(token.Comma) {
			tok, err := p.expect(token.Ident)
			if err != nil {
				return nil, err
			}
			targets = append(targets, ast.AssignTarget{Pos: tok.Pos, Name: tok.Lexeme})
		}
		if _, err := p.expect(token.Assign); err != nil {
			return nil, err
		}
		value, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		return &ast.AssignStmt{
			Start:   firstIdent.Start,
			Name:    firstIdent.Name,
			Targets: targets,
			Value:   value,
		}, nil
	}

	if p.match(token.Assign) {
		value, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}

		switch target := expr.(type) {
		case *ast.IdentExpr:
			return &ast.AssignStmt{
				Start:   target.Start,
				Name:    target.Name,
				Targets: []ast.AssignTarget{{Pos: target.Start, Name: target.Name}},
				Value:   value,
			}, nil
		case *ast.IndexExpr:
			return &ast.IndexAssignStmt{Start: target.Start, Target: target, Value: value}, nil
		case *ast.FieldExpr:
			return &ast.FieldAssignStmt{Start: target.Start, Target: target, Value: value}, nil
		case *ast.UnaryExpr:
			if target.Operator == "*" {
				return &ast.DerefAssignStmt{Start: target.Start, Target: target, Value: value}, nil
			}
			return nil, p.errorf(token.Token{Pos: expr.Pos()}, "invalid assignment target")
		default:
			return nil, p.errorf(token.Token{Pos: expr.Pos()}, "invalid assignment target")
		}
	}

	return &ast.ExprStmt{Expr: expr}, nil
}

func (p *Parser) parseExpression(minPrecedence int) (ast.Expression, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for {
		if p.current().Type == token.Asterisk && p.pos > 0 && p.current().Pos.Line > p.tokens[p.pos-1].Pos.Line {
			return left, nil
		}
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

func (p *Parser) parseUnary() (ast.Expression, error) {
	if p.check(token.Minus) || p.check(token.Not) || p.check(token.Asterisk) || p.check(token.Ampersand) {
		operator := p.advance()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpr{Start: operator.Pos, Operator: operator.Lexeme, Operand: operand}, nil
	}
	return p.parsePostfix()
}

func (p *Parser) parsePostfix() (ast.Expression, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		if p.match(token.Dot) {
			member, err := p.expect(token.Ident)
			if err != nil {
				return nil, err
			}
			expr = &ast.FieldExpr{
				Start:  expr.Pos(),
				Object: expr,
				Field:  member.Lexeme,
			}
			continue
		}
		if p.match(token.LBracket) {
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
			continue
		}
		break
	}

	return expr, nil
}

func (p *Parser) isStructLiteralBraceAt(bracePos int) bool {
	if bracePos < 0 || bracePos >= len(p.tokens) || p.tokens[bracePos].Type != token.LBrace {
		return false
	}
	if bracePos+2 < len(p.tokens) && p.tokens[bracePos+1].Type == token.Ident && p.tokens[bracePos+2].Type == token.Colon {
		return true
	}
	if bracePos+1 < len(p.tokens) && p.tokens[bracePos+1].Type == token.RBrace {
		if !p.inConditionHeader {
			return true
		}
		if bracePos+2 < len(p.tokens) {
			switch p.tokens[bracePos+2].Type {
			case token.LBrace, token.Equal, token.NotEqual, token.Dot, token.Semicolon:
				return true
			}
		}
	}
	return false
}

func (p *Parser) parsePrimary() (ast.Expression, error) {
	switch {
	case p.check(token.LBracket), p.check(token.List), p.check(token.Map):
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
	case p.check(token.Nil):
		tok := p.advance()
		return &ast.NilLiteral{Start: tok.Pos}, nil
	case p.check(token.Ident):
		ident := p.advance()
		if ident.Lexeme == "new" && p.match(token.LParen) {
			typ, err := p.parseType()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(token.RParen); err != nil {
				return nil, err
			}
			return &ast.NewExpr{Start: ident.Pos, ElemType: typ}, nil
		}
		if ident.Lexeme == "make" && p.match(token.LParen) {
			return p.parseMakeExpr(ident)
		}
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
		if p.check(token.LBrace) && p.isStructLiteralBraceAt(p.pos) {
			return p.parseStructLiteral(ident.Pos, &ast.StructType{Name: ident.Lexeme})
		}
		if p.check(token.Dot) && p.peek(1).Type == token.Ident {
			if p.peek(2).Type == token.LParen {
				p.advance()
				member := p.advance()
				p.advance()
				args, err := p.parseArguments()
				if err != nil {
					return nil, err
				}
				if _, err := p.expect(token.RParen); err != nil {
					return nil, err
				}
				return &ast.CallExpr{Start: ident.Pos, Package: ident.Lexeme, Callee: member.Lexeme, Args: args}, nil
			}
			if p.peek(2).Type == token.LBrace && p.isStructLiteralBraceAt(p.pos+2) {
				p.advance()
				member := p.advance()
				return p.parseStructLiteral(ident.Pos, &ast.StructType{Package: ident.Lexeme, Name: member.Lexeme})
			}
		}
		return &ast.IdentExpr{Start: ident.Pos, Name: ident.Lexeme}, nil
	case p.match(token.LParen):
		prevCond := p.inConditionHeader
		p.inConditionHeader = false
		expr, err := p.parseExpression(0)
		p.inConditionHeader = prevCond
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

func (p *Parser) parseStructLiteral(start token.Position, structType *ast.StructType) (ast.Expression, error) {
	if _, err := p.expect(token.LBrace); err != nil {
		return nil, err
	}
	prevCond := p.inConditionHeader
	p.inConditionHeader = false
	defer func() {
		p.inConditionHeader = prevCond
	}()

	var fields []ast.StructFieldInit
	for !p.check(token.RBrace) && !p.check(token.EOF) {
		fieldTok, err := p.expect(token.Ident)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(token.Colon); err != nil {
			return nil, err
		}
		val, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		fields = append(fields, ast.StructFieldInit{
			Pos:   fieldTok.Pos,
			Name:  fieldTok.Lexeme,
			Value: val,
		})
		if !p.match(token.Comma) && !p.match(token.Semicolon) {
			break
		}
	}
	if _, err := p.expect(token.RBrace); err != nil {
		return nil, err
	}
	return &ast.StructLiteral{
		Start:  start,
		Type:   structType,
		Fields: fields,
	}, nil
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
	prevCond := p.inConditionHeader
	p.inConditionHeader = false
	defer func() {
		p.inConditionHeader = prevCond
	}()

	if mapType, ok := typ.(*ast.MapType); ok {
		var entries []ast.MapEntry
		for !p.check(token.RBrace) && !p.check(token.EOF) {
			key, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(token.Colon); err != nil {
				return nil, err
			}
			val, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			entries = append(entries, ast.MapEntry{
				Pos:   key.Pos(),
				Key:   key,
				Value: val,
			})
			if !p.match(token.Comma) {
				break
			}
		}
		if _, err := p.expect(token.RBrace); err != nil {
			return nil, err
		}
		return &ast.MapLiteral{
			Start:   start,
			Type:    mapType,
			Entries: entries,
		}, nil
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
	if _, isMap := typ.(*ast.MapType); isMap && p.match(token.RParen) {
		return &ast.MakeExpr{Start: ident.Pos, Type: typ, Len: nil}, nil
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
	case token.Or:
		return 0
	case token.And:
		return 1
	case token.Equal, token.NotEqual, token.Less, token.LessEqual, token.Greater, token.GreaterEqual, token.In:
		return 2
	case token.Plus, token.Minus:
		return 3
	case token.Asterisk, token.Slash, token.Percent:
		return 4
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

func (p *Parser) peek(offset int) token.Token {
	idx := p.pos + offset
	if idx < 0 {
		return p.tokens[0]
	}
	if idx >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[idx]
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
	case token.Assign, token.Equal, token.NotEqual, token.Less, token.LessEqual, token.Greater, token.GreaterEqual,
		token.Plus, token.Minus, token.Asterisk, token.Slash, token.Percent, token.And, token.Or, token.Not, token.Ampersand,
		token.Comma, token.Colon, token.Dot, token.Semicolon, token.LParen, token.RParen, token.LBrace, token.RBrace, token.LBracket, token.RBracket:
		return true
	default:
		return false
	}
}
