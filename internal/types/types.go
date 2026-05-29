package types

import (
	"fmt"

	"github.com/claudioscheer/trux/internal/ast"
	"github.com/claudioscheer/trux/internal/token"
)

type Error struct {
	Pos token.Position
	Msg string
}

func (e *Error) Error() string {
	return fmt.Sprintf("type error at %d:%d: %s", e.Pos.Line, e.Pos.Column, e.Msg)
}

type FuncSig struct {
	Name       string
	Params     []ast.Param
	ReturnType ast.Type
}

type Info struct {
	Funcs         map[string]FuncSig
	Locals        map[*ast.FuncDecl]map[string]ast.Type
	ExprTypes     map[ast.Expression]ast.Type
	ResolvedCalls map[*ast.CallExpr]FuncSig
	PrintCalls    map[*ast.CallExpr][]ast.Type
}

type checker struct {
	info *Info
}

func Check(program *ast.Program) (*Info, error) {
	c := &checker{
		info: &Info{
			Funcs:         map[string]FuncSig{},
			Locals:        map[*ast.FuncDecl]map[string]ast.Type{},
			ExprTypes:     map[ast.Expression]ast.Type{},
			ResolvedCalls: map[*ast.CallExpr]FuncSig{},
			PrintCalls:    map[*ast.CallExpr][]ast.Type{},
		},
	}

	if err := c.collectFunctions(program); err != nil {
		return nil, err
	}
	if err := c.checkMain(program); err != nil {
		return nil, err
	}
	for _, fn := range program.Functions {
		if err := c.checkFunc(fn); err != nil {
			return nil, err
		}
	}

	return c.info, nil
}

func (c *checker) collectFunctions(program *ast.Program) error {
	for _, fn := range program.Functions {
		if _, exists := c.info.Funcs[fn.Name]; exists {
			return typeError(fn.Pos, "duplicate function %q", fn.Name)
		}

		c.info.Funcs[fn.Name] = FuncSig{
			Name:       fn.Name,
			Params:     fn.Params,
			ReturnType: fn.ReturnType,
		}
	}

	return nil
}

func (c *checker) checkMain(program *ast.Program) error {
	sig, ok := c.info.Funcs["main"]
	if !ok {
		pos := token.Position{Line: 1, Column: 1}
		if len(program.Functions) > 0 {
			pos = program.Functions[0].Pos
		}
		return typeError(pos, "missing main function")
	}
	if len(sig.Params) != 0 {
		return typeError(findFunc(program, "main").Pos, "main must not have parameters")
	}
	if sig.ReturnType != ast.IntType {
		return typeError(findFunc(program, "main").Pos, "main must return int")
	}

	return nil
}

func (c *checker) checkFunc(fn *ast.FuncDecl) error {
	locals := map[string]ast.Type{}
	c.info.Locals[fn] = locals

	for _, param := range fn.Params {
		if _, exists := locals[param.Name]; exists {
			return typeError(fn.Pos, "duplicate parameter %q", param.Name)
		}
		locals[param.Name] = param.Type
	}

	for _, stmt := range fn.Body.Statements {
		if err := c.checkStmt(fn, locals, stmt); err != nil {
			return err
		}
	}

	return nil
}

func (c *checker) checkStmt(fn *ast.FuncDecl, locals map[string]ast.Type, stmt ast.Statement) error {
	switch stmt := stmt.(type) {
	case *ast.LetStmt:
		if _, exists := locals[stmt.Name]; exists {
			return typeError(stmt.Start, "duplicate local variable %q", stmt.Name)
		}

		valueType, err := c.checkExpr(locals, stmt.Value, false)
		if err != nil {
			return err
		}
		if valueType != stmt.Type {
			return typeError(stmt.Value.Pos(), "cannot assign %s to %s", valueType, stmt.Type)
		}

		locals[stmt.Name] = stmt.Type
		return nil
	case *ast.ReturnStmt:
		valueType, err := c.checkExpr(locals, stmt.Value, false)
		if err != nil {
			return err
		}
		if valueType != fn.ReturnType {
			return typeError(stmt.Value.Pos(), "cannot return %s from function returning %s", valueType, fn.ReturnType)
		}
		return nil
	case *ast.ExprStmt:
		_, err := c.checkExpr(locals, stmt.Expr, true)
		return err
	default:
		return typeError(stmt.Pos(), "unsupported statement %T", stmt)
	}
}

func (c *checker) checkExpr(locals map[string]ast.Type, expr ast.Expression, allowPrint bool) (ast.Type, error) {
	switch expr := expr.(type) {
	case *ast.IntLiteral:
		c.info.ExprTypes[expr] = ast.IntType
		return ast.IntType, nil
	case *ast.StringLiteral:
		c.info.ExprTypes[expr] = ast.StringType
		return ast.StringType, nil
	case *ast.BoolLiteral:
		c.info.ExprTypes[expr] = ast.BoolType
		return ast.BoolType, nil
	case *ast.IdentExpr:
		typ, ok := locals[expr.Name]
		if !ok {
			return "", typeError(expr.Start, "undefined variable %q", expr.Name)
		}
		c.info.ExprTypes[expr] = typ
		return typ, nil
	case *ast.BinaryExpr:
		leftType, err := c.checkExpr(locals, expr.Left, false)
		if err != nil {
			return "", err
		}
		rightType, err := c.checkExpr(locals, expr.Right, false)
		if err != nil {
			return "", err
		}
		if leftType != ast.IntType || rightType != ast.IntType {
			return "", typeError(expr.Start, "operator %q requires int operands", expr.Operator)
		}
		c.info.ExprTypes[expr] = ast.IntType
		return ast.IntType, nil
	case *ast.CallExpr:
		return c.checkCall(locals, expr, allowPrint)
	default:
		return "", typeError(expr.Pos(), "unsupported expression %T", expr)
	}
}

func (c *checker) checkCall(locals map[string]ast.Type, expr *ast.CallExpr, allowPrint bool) (ast.Type, error) {
	argTypes := make([]ast.Type, 0, len(expr.Args))
	for _, arg := range expr.Args {
		argType, err := c.checkExpr(locals, arg, false)
		if err != nil {
			return "", err
		}
		argTypes = append(argTypes, argType)
	}

	if expr.Callee == "print" {
		if !allowPrint {
			return "", typeError(expr.Start, "print can only be used as a statement")
		}
		if len(argTypes) == 0 {
			return "", typeError(expr.Start, "print expects at least 1 argument, got 0")
		}
		c.info.PrintCalls[expr] = argTypes
		c.info.ExprTypes[expr] = argTypes[len(argTypes)-1]
		return argTypes[len(argTypes)-1], nil
	}

	sig, ok := c.info.Funcs[expr.Callee]
	if !ok {
		return "", typeError(expr.Start, "undefined function %q", expr.Callee)
	}
	if len(argTypes) != len(sig.Params) {
		return "", typeError(expr.Start, "%s expects %d arguments, got %d", expr.Callee, len(sig.Params), len(argTypes))
	}
	for i, argType := range argTypes {
		want := sig.Params[i].Type
		if argType != want {
			return "", typeError(expr.Args[i].Pos(), "argument %d to %s has type %s, want %s", i+1, expr.Callee, argType, want)
		}
	}

	c.info.ResolvedCalls[expr] = sig
	c.info.ExprTypes[expr] = sig.ReturnType
	return sig.ReturnType, nil
}

func typeError(pos token.Position, format string, args ...any) error {
	return &Error{
		Pos: pos,
		Msg: fmt.Sprintf(format, args...),
	}
}

func findFunc(program *ast.Program, name string) *ast.FuncDecl {
	for _, fn := range program.Functions {
		if fn.Name == name {
			return fn
		}
	}

	return nil
}
