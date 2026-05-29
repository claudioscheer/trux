package ir

import (
	"fmt"

	"github.com/claudioscheer/trux/internal/ast"
	"github.com/claudioscheer/trux/internal/types"
)

type Program struct {
	PackageName string
	Functions   []*Func
}

type Func struct {
	Name       string
	Params     []Param
	ReturnType ast.Type
	Body       []Stmt
}

type Param struct {
	Name string
	Type ast.Type
}

type Stmt interface {
	stmtNode()
}

type LetStmt struct {
	Name  string
	Type  ast.Type
	Value Expr
}

func (*LetStmt) stmtNode() {}

type ReturnStmt struct {
	Value Expr
}

func (*ReturnStmt) stmtNode() {}

type AssignStmt struct {
	Name  string
	Value Expr
}

func (*AssignStmt) stmtNode() {}

type IfStmt struct {
	Condition Expr
	Then      []Stmt
	Else      []Stmt
}

func (*IfStmt) stmtNode() {}

type WhileStmt struct {
	Condition Expr
	Body      []Stmt
}

func (*WhileStmt) stmtNode() {}

type PrintStmt struct {
	Args  []Expr
	Types []ast.Type
}

func (*PrintStmt) stmtNode() {}

type ExprStmt struct {
	Expr Expr
}

func (*ExprStmt) stmtNode() {}

type Expr interface {
	Type() ast.Type
	exprNode()
}

type IdentExpr struct {
	Name string
	Typ  ast.Type
}

func (e *IdentExpr) Type() ast.Type { return e.Typ }

func (*IdentExpr) exprNode() {}

type IntLiteral struct {
	Value string
	Typ   ast.Type
}

func (e *IntLiteral) Type() ast.Type { return e.Typ }

func (*IntLiteral) exprNode() {}

type FloatLiteral struct {
	Value string
	Typ   ast.Type
}

func (e *FloatLiteral) Type() ast.Type { return e.Typ }

func (*FloatLiteral) exprNode() {}

type StringLiteral struct {
	Value string
	Typ   ast.Type
}

func (e *StringLiteral) Type() ast.Type { return e.Typ }

func (*StringLiteral) exprNode() {}

type BoolLiteral struct {
	Value bool
	Typ   ast.Type
}

func (e *BoolLiteral) Type() ast.Type { return e.Typ }

func (*BoolLiteral) exprNode() {}

type CallExpr struct {
	Callee     string
	ReturnType ast.Type
	Args       []Expr
}

func (e *CallExpr) Type() ast.Type { return e.ReturnType }

func (*CallExpr) exprNode() {}

type BinaryExpr struct {
	Left     Expr
	Operator string
	Right    Expr
	Typ      ast.Type
}

func (e *BinaryExpr) Type() ast.Type { return e.Typ }

func (*BinaryExpr) exprNode() {}

func Build(program *ast.Program, info *types.Info) (*Program, error) {
	builder := builder{info: info}
	return builder.buildProgram(program)
}

type builder struct {
	info *types.Info
}

func (b builder) buildProgram(program *ast.Program) (*Program, error) {
	out := &Program{PackageName: program.PackageName}
	for _, fn := range program.Functions {
		irFn, err := b.buildFunc(fn)
		if err != nil {
			return nil, err
		}
		out.Functions = append(out.Functions, irFn)
	}

	return out, nil
}

func (b builder) buildFunc(fn *ast.FuncDecl) (*Func, error) {
	out := &Func{
		Name:       fn.Name,
		ReturnType: fn.ReturnType,
	}
	for _, param := range fn.Params {
		out.Params = append(out.Params, Param{Name: param.Name, Type: param.Type})
	}
	for _, stmt := range fn.Body.Statements {
		irStmt, err := b.buildStmt(stmt)
		if err != nil {
			return nil, err
		}
		out.Body = append(out.Body, irStmt)
	}

	return out, nil
}

func (b builder) buildStmt(stmt ast.Statement) (Stmt, error) {
	switch stmt := stmt.(type) {
	case *ast.LetStmt:
		value, err := b.buildExpr(stmt.Value)
		if err != nil {
			return nil, err
		}
		return &LetStmt{Name: stmt.Name, Type: stmt.Type, Value: value}, nil
	case *ast.ReturnStmt:
		value, err := b.buildExpr(stmt.Value)
		if err != nil {
			return nil, err
		}
		return &ReturnStmt{Value: value}, nil
	case *ast.AssignStmt:
		value, err := b.buildExpr(stmt.Value)
		if err != nil {
			return nil, err
		}
		return &AssignStmt{Name: stmt.Name, Value: value}, nil
	case *ast.IfStmt:
		condition, err := b.buildExpr(stmt.Condition)
		if err != nil {
			return nil, err
		}
		thenStmts, err := b.buildStmts(stmt.Then.Statements)
		if err != nil {
			return nil, err
		}
		var elseStmts []Stmt
		if stmt.Else != nil {
			elseStmts, err = b.buildStmts(stmt.Else.Statements)
			if err != nil {
				return nil, err
			}
		}
		return &IfStmt{Condition: condition, Then: thenStmts, Else: elseStmts}, nil
	case *ast.WhileStmt:
		condition, err := b.buildExpr(stmt.Condition)
		if err != nil {
			return nil, err
		}
		body, err := b.buildStmts(stmt.Body.Statements)
		if err != nil {
			return nil, err
		}
		return &WhileStmt{Condition: condition, Body: body}, nil
	case *ast.ExprStmt:
		if call, ok := stmt.Expr.(*ast.CallExpr); ok {
			if printTypes, ok := b.info.PrintCalls[call]; ok {
				args, err := b.buildExprs(call.Args)
				if err != nil {
					return nil, err
				}
				return &PrintStmt{Args: args, Types: printTypes}, nil
			}
		}

		expr, err := b.buildExpr(stmt.Expr)
		if err != nil {
			return nil, err
		}
		return &ExprStmt{Expr: expr}, nil
	default:
		return nil, fmt.Errorf("unsupported AST statement %T", stmt)
	}
}

func (b builder) buildExpr(expr ast.Expression) (Expr, error) {
	typ, ok := b.info.ExprTypes[expr]
	if !ok {
		return nil, fmt.Errorf("missing type for AST expression %T", expr)
	}

	switch expr := expr.(type) {
	case *ast.IntLiteral:
		return &IntLiteral{Value: expr.Value, Typ: typ}, nil
	case *ast.FloatLiteral:
		return &FloatLiteral{Value: expr.Value, Typ: typ}, nil
	case *ast.StringLiteral:
		return &StringLiteral{Value: expr.Value, Typ: typ}, nil
	case *ast.BoolLiteral:
		return &BoolLiteral{Value: expr.Value, Typ: typ}, nil
	case *ast.IdentExpr:
		return &IdentExpr{Name: expr.Name, Typ: typ}, nil
	case *ast.CallExpr:
		sig, ok := b.info.ResolvedCalls[expr]
		if !ok {
			return nil, fmt.Errorf("missing resolved call for %q", expr.Callee)
		}
		args, err := b.buildExprs(expr.Args)
		if err != nil {
			return nil, err
		}
		return &CallExpr{Callee: sig.Name, ReturnType: sig.ReturnType, Args: args}, nil
	case *ast.BinaryExpr:
		left, err := b.buildExpr(expr.Left)
		if err != nil {
			return nil, err
		}
		right, err := b.buildExpr(expr.Right)
		if err != nil {
			return nil, err
		}
		return &BinaryExpr{Left: left, Operator: expr.Operator, Right: right, Typ: typ}, nil
	default:
		return nil, fmt.Errorf("unsupported AST expression %T", expr)
	}
}

func (b builder) buildStmts(stmts []ast.Statement) ([]Stmt, error) {
	out := make([]Stmt, 0, len(stmts))
	for _, stmt := range stmts {
		irStmt, err := b.buildStmt(stmt)
		if err != nil {
			return nil, err
		}
		out = append(out, irStmt)
	}

	return out, nil
}

func (b builder) buildExprs(exprs []ast.Expression) ([]Expr, error) {
	out := make([]Expr, 0, len(exprs))
	for _, expr := range exprs {
		irExpr, err := b.buildExpr(expr)
		if err != nil {
			return nil, err
		}
		out = append(out, irExpr)
	}

	return out, nil
}
