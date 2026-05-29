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
	case *ast.AssignStmt:
		varType, ok := locals[stmt.Name]
		if !ok {
			return typeError(stmt.Start, "undefined variable %q", stmt.Name)
		}
		valueType, err := c.checkExpr(locals, stmt.Value, false)
		if err != nil {
			return err
		}
		if valueType != varType {
			return typeError(stmt.Value.Pos(), "cannot assign %s to %s", valueType, varType)
		}
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
	case *ast.IfStmt:
		conditionType, err := c.checkExpr(locals, stmt.Condition, false)
		if err != nil {
			return err
		}
		if conditionType != ast.BoolType {
			return typeError(stmt.Condition.Pos(), "if condition must be bool, got %s", conditionType)
		}
		for _, inner := range stmt.Then.Statements {
			if err := c.checkStmt(fn, locals, inner); err != nil {
				return err
			}
		}
		if stmt.Else != nil {
			for _, inner := range stmt.Else.Statements {
				if err := c.checkStmt(fn, locals, inner); err != nil {
					return err
				}
			}
		}
		return nil
	case *ast.WhileStmt:
		conditionType, err := c.checkExpr(locals, stmt.Condition, false)
		if err != nil {
			return err
		}
		if conditionType != ast.BoolType {
			return typeError(stmt.Condition.Pos(), "while condition must be bool, got %s", conditionType)
		}
		for _, inner := range stmt.Body.Statements {
			if err := c.checkStmt(fn, locals, inner); err != nil {
				return err
			}
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
	case *ast.FloatLiteral:
		c.info.ExprTypes[expr] = ast.FloatType
		return ast.FloatType, nil
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
		typ, err := binaryType(expr.Start, expr.Operator, leftType, rightType)
		if err != nil {
			return "", err
		}
		c.info.ExprTypes[expr] = typ
		return typ, nil
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

func binaryType(pos token.Position, operator string, leftType ast.Type, rightType ast.Type) (ast.Type, error) {
	switch operator {
	case "+", "-", "*", "/":
		if leftType == ast.IntType && rightType == ast.IntType {
			return ast.IntType, nil
		}
		if leftType == ast.FloatType && rightType == ast.FloatType {
			return ast.FloatType, nil
		}
		return "", typeError(pos, "operator %q requires matching numeric operands, got %s and %s", operator, leftType, rightType)
	case "==", "!=":
		if leftType == rightType && comparableType(leftType) {
			return ast.BoolType, nil
		}
		return "", typeError(pos, "operator %q requires matching comparable operands, got %s and %s", operator, leftType, rightType)
	case "<", "<=", ">", ">=":
		if leftType == rightType && (leftType == ast.IntType || leftType == ast.FloatType) {
			return ast.BoolType, nil
		}
		return "", typeError(pos, "operator %q requires matching numeric operands, got %s and %s", operator, leftType, rightType)
	case "in":
		if leftType == ast.StringType && rightType == ast.StringType {
			return ast.BoolType, nil
		}
		return "", typeError(pos, "operator %q requires string operands, got %s and %s", operator, leftType, rightType)
	default:
		return "", typeError(pos, "unsupported operator %q", operator)
	}
}

func comparableType(typ ast.Type) bool {
	switch typ {
	case ast.IntType, ast.FloatType, ast.StringType, ast.BoolType:
		return true
	default:
		return false
	}
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
