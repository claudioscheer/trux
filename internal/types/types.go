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
	ExprOrigins   map[ast.Expression]Origin
	ResolvedCalls map[*ast.CallExpr]FuncSig
	PrintCalls    map[*ast.CallExpr][]ast.Type
	LenCalls      map[*ast.CallExpr]ast.Type
	AppendCalls   map[*ast.CallExpr]AppendSig
	CloneCalls    map[*ast.CallExpr]CloneSig
}

type AppendSig struct {
	ListType ast.Type
	ElemType ast.Type
}

type CloneSig struct {
	Type ast.Type
}

type Origin string

const (
	OriginBorrowed   Origin = "borrowed"
	OriginScratch    Origin = "scratch"
	OriginOwned      Origin = "owned"
	OriginFrameOwned Origin = "frame-owned"
	OriginUnknown    Origin = "unknown"
)

type checker struct {
	info *Info
}

type localInfo struct {
	typ    ast.Type
	origin Origin
}

type scope struct {
	locals map[string]localInfo
	parent *scope
}

func newScope(parent *scope) *scope {
	return &scope{
		locals: map[string]localInfo{},
		parent: parent,
	}
}

func (s *scope) lookup(name string) (localInfo, bool) {
	for scope := s; scope != nil; scope = scope.parent {
		if local, ok := scope.locals[name]; ok {
			return local, true
		}
	}

	return localInfo{}, false
}

func (s *scope) assign(name string, local localInfo) bool {
	for scope := s; scope != nil; scope = scope.parent {
		if _, ok := scope.locals[name]; ok {
			scope.locals[name] = local
			return true
		}
	}

	return false
}

func Check(program *ast.Program) (*Info, error) {
	c := &checker{
		info: &Info{
			Funcs:         map[string]FuncSig{},
			Locals:        map[*ast.FuncDecl]map[string]ast.Type{},
			ExprTypes:     map[ast.Expression]ast.Type{},
			ExprOrigins:   map[ast.Expression]Origin{},
			ResolvedCalls: map[*ast.CallExpr]FuncSig{},
			PrintCalls:    map[*ast.CallExpr][]ast.Type{},
			LenCalls:      map[*ast.CallExpr]ast.Type{},
			AppendCalls:   map[*ast.CallExpr]AppendSig{},
			CloneCalls:    map[*ast.CallExpr]CloneSig{},
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
		if err := validateType(fn.Pos, fn.ReturnType); err != nil {
			return err
		}
		for _, param := range fn.Params {
			if err := validateType(fn.Pos, param.Type); err != nil {
				return err
			}
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
	if !ast.TypeEqual(sig.ReturnType, ast.IntType) {
		return typeError(findFunc(program, "main").Pos, "main must return int")
	}

	return nil
}

func (c *checker) checkFunc(fn *ast.FuncDecl) error {
	c.info.Locals[fn] = map[string]ast.Type{}
	locals := newScope(nil)

	for _, param := range fn.Params {
		if _, exists := c.info.Locals[fn][param.Name]; exists {
			return typeError(fn.Pos, "duplicate parameter %q", param.Name)
		}
		c.info.Locals[fn][param.Name] = param.Type
		locals.locals[param.Name] = localInfo{
			typ:    param.Type,
			origin: originForParam(param.Type),
		}
	}

	for _, stmt := range fn.Body.Statements {
		if err := c.checkStmt(fn, locals, stmt); err != nil {
			return err
		}
	}

	return nil
}

func (c *checker) checkStmt(fn *ast.FuncDecl, locals *scope, stmt ast.Statement) error {
	switch stmt := stmt.(type) {
	case *ast.LetStmt:
		if _, exists := c.info.Locals[fn][stmt.Name]; exists {
			return typeError(stmt.Start, "duplicate local variable %q", stmt.Name)
		}
		if err := validateType(stmt.Start, stmt.Type); err != nil {
			return err
		}

		valueType, err := c.checkExpr(locals, stmt.Value, false)
		if err != nil {
			return err
		}
		if !ast.TypeEqual(valueType, stmt.Type) {
			return typeError(stmt.Value.Pos(), "cannot assign %s to %s", valueType, stmt.Type)
		}

		c.info.Locals[fn][stmt.Name] = stmt.Type
		locals.locals[stmt.Name] = localInfo{
			typ:    stmt.Type,
			origin: c.exprOrigin(stmt.Value),
		}
		return nil
	case *ast.AssignStmt:
		local, ok := locals.lookup(stmt.Name)
		if !ok {
			return typeError(stmt.Start, "undefined variable %q", stmt.Name)
		}
		valueType, err := c.checkExpr(locals, stmt.Value, false)
		if err != nil {
			return err
		}
		if !ast.TypeEqual(valueType, local.typ) {
			return typeError(stmt.Value.Pos(), "cannot assign %s to %s", valueType, local.typ)
		}
		local.origin = c.exprOrigin(stmt.Value)
		locals.assign(stmt.Name, local)
		return nil
	case *ast.IndexAssignStmt:
		collectionType, err := c.checkExpr(locals, stmt.Target.Collection, false)
		if err != nil {
			return err
		}
		indexType, err := c.checkExpr(locals, stmt.Target.Index, false)
		if err != nil {
			return err
		}
		if !ast.TypeEqual(indexType, ast.IntType) {
			return typeError(stmt.Target.Index.Pos(), "index must be int, got %s", indexType)
		}
		if ast.TypeEqual(collectionType, ast.StringType) {
			return typeError(stmt.Target.Pos(), "cannot assign through string index")
		}
		elemType, ok := ast.ElementType(collectionType)
		if !ok {
			return typeError(stmt.Target.Collection.Pos(), "cannot index %s", collectionType)
		}
		if err := c.checkCanMutate(stmt.Target.Collection); err != nil {
			return err
		}
		valueType, err := c.checkExpr(locals, stmt.Value, false)
		if err != nil {
			return err
		}
		if !ast.TypeEqual(valueType, elemType) {
			return typeError(stmt.Value.Pos(), "cannot assign %s to %s element", valueType, elemType)
		}
		c.setExpr(stmt.Target, elemType, indexOrigin(collectionType, c.exprOrigin(stmt.Target.Collection)))
		return nil
	case *ast.ReturnStmt:
		valueType, err := c.checkExpr(locals, stmt.Value, false)
		if err != nil {
			return err
		}
		if !ast.TypeEqual(valueType, fn.ReturnType) {
			return typeError(stmt.Value.Pos(), "cannot return %s from function returning %s", valueType, fn.ReturnType)
		}
		return nil
	case *ast.IfStmt:
		conditionType, err := c.checkExpr(locals, stmt.Condition, false)
		if err != nil {
			return err
		}
		if !ast.TypeEqual(conditionType, ast.BoolType) {
			return typeError(stmt.Condition.Pos(), "if condition must be bool, got %s", conditionType)
		}
		thenScope := newScope(locals)
		for _, inner := range stmt.Then.Statements {
			if err := c.checkStmt(fn, thenScope, inner); err != nil {
				return err
			}
		}
		if stmt.Else != nil {
			elseScope := newScope(locals)
			for _, inner := range stmt.Else.Statements {
				if err := c.checkStmt(fn, elseScope, inner); err != nil {
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
		if !ast.TypeEqual(conditionType, ast.BoolType) {
			return typeError(stmt.Condition.Pos(), "while condition must be bool, got %s", conditionType)
		}
		bodyScope := newScope(locals)
		for _, inner := range stmt.Body.Statements {
			if err := c.checkStmt(fn, bodyScope, inner); err != nil {
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

func (c *checker) checkExpr(locals *scope, expr ast.Expression, allowPrint bool) (ast.Type, error) {
	switch expr := expr.(type) {
	case *ast.IntLiteral:
		c.setExpr(expr, ast.IntType, OriginOwned)
		return ast.IntType, nil
	case *ast.FloatLiteral:
		c.setExpr(expr, ast.FloatType, OriginOwned)
		return ast.FloatType, nil
	case *ast.StringLiteral:
		c.setExpr(expr, ast.StringType, OriginOwned)
		return ast.StringType, nil
	case *ast.BoolLiteral:
		c.setExpr(expr, ast.BoolType, OriginOwned)
		return ast.BoolType, nil
	case *ast.ArrayLiteral:
		if err := validateType(expr.Start, expr.Type); err != nil {
			return nil, err
		}
		arrayType, ok := expr.Type.(*ast.ArrayType)
		if !ok {
			return nil, typeError(expr.Start, "array literal must have array type")
		}
		if len(expr.Elements) != arrayType.Length {
			return nil, typeError(expr.Start, "array literal for %s has %d elements, want %d", expr.Type, len(expr.Elements), arrayType.Length)
		}
		for i, elem := range expr.Elements {
			elemType, err := c.checkExpr(locals, elem, false)
			if err != nil {
				return nil, err
			}
			if !ast.TypeEqual(elemType, arrayType.Elem) {
				return nil, typeError(elem.Pos(), "array element %d has type %s, want %s", i+1, elemType, arrayType.Elem)
			}
		}
		c.setExpr(expr, expr.Type, OriginScratch)
		return expr.Type, nil
	case *ast.ListLiteral:
		if err := validateType(expr.Start, expr.Type); err != nil {
			return nil, err
		}
		listType, ok := expr.Type.(*ast.ListType)
		if !ok {
			return nil, typeError(expr.Start, "list literal must have list type")
		}
		for i, elem := range expr.Elements {
			elemType, err := c.checkExpr(locals, elem, false)
			if err != nil {
				return nil, err
			}
			if !ast.TypeEqual(elemType, listType.Elem) {
				return nil, typeError(elem.Pos(), "list element %d has type %s, want %s", i+1, elemType, listType.Elem)
			}
		}
		c.setExpr(expr, expr.Type, OriginScratch)
		return expr.Type, nil
	case *ast.MakeExpr:
		if err := validateType(expr.Start, expr.Type); err != nil {
			return nil, err
		}
		if _, ok := expr.Type.(*ast.SliceType); !ok {
			return nil, typeError(expr.Start, "make expects slice type, got %s", expr.Type)
		}
		lenType, err := c.checkExpr(locals, expr.Len, false)
		if err != nil {
			return nil, err
		}
		if !ast.TypeEqual(lenType, ast.IntType) {
			return nil, typeError(expr.Len.Pos(), "make length must be int, got %s", lenType)
		}
		c.setExpr(expr, expr.Type, OriginScratch)
		return expr.Type, nil
	case *ast.IdentExpr:
		local, ok := locals.lookup(expr.Name)
		if !ok {
			return nil, typeError(expr.Start, "undefined variable %q", expr.Name)
		}
		c.setExpr(expr, local.typ, local.origin)
		return local.typ, nil
	case *ast.BinaryExpr:
		leftType, err := c.checkExpr(locals, expr.Left, false)
		if err != nil {
			return nil, err
		}
		rightType, err := c.checkExpr(locals, expr.Right, false)
		if err != nil {
			return nil, err
		}
		typ, err := binaryType(expr.Start, expr.Operator, leftType, rightType)
		if err != nil {
			return nil, err
		}
		c.setExpr(expr, typ, binaryOrigin(expr.Operator, typ))
		return typ, nil
	case *ast.CallExpr:
		return c.checkCall(locals, expr, allowPrint)
	case *ast.IndexExpr:
		collectionType, err := c.checkExpr(locals, expr.Collection, false)
		if err != nil {
			return nil, err
		}
		indexType, err := c.checkExpr(locals, expr.Index, false)
		if err != nil {
			return nil, err
		}
		if !ast.TypeEqual(indexType, ast.IntType) {
			return nil, typeError(expr.Index.Pos(), "index must be int, got %s", indexType)
		}
		var typ ast.Type
		if ast.TypeEqual(collectionType, ast.StringType) {
			typ = ast.StringType
		} else {
			elemType, ok := ast.ElementType(collectionType)
			if !ok {
				return nil, typeError(expr.Collection.Pos(), "cannot index %s", collectionType)
			}
			typ = elemType
		}
		c.setExpr(expr, typ, indexOrigin(collectionType, c.exprOrigin(expr.Collection)))
		return typ, nil
	case *ast.SliceExpr:
		collectionType, err := c.checkExpr(locals, expr.Collection, false)
		if err != nil {
			return nil, err
		}
		if expr.StartIndex != nil {
			startType, err := c.checkExpr(locals, expr.StartIndex, false)
			if err != nil {
				return nil, err
			}
			if !ast.TypeEqual(startType, ast.IntType) {
				return nil, typeError(expr.StartIndex.Pos(), "slice start must be int, got %s", startType)
			}
		}
		if expr.EndIndex != nil {
			endType, err := c.checkExpr(locals, expr.EndIndex, false)
			if err != nil {
				return nil, err
			}
			if !ast.TypeEqual(endType, ast.IntType) {
				return nil, typeError(expr.EndIndex.Pos(), "slice end must be int, got %s", endType)
			}
		}
		var typ ast.Type
		if ast.TypeEqual(collectionType, ast.StringType) {
			typ = ast.StringType
		} else {
			elemType, ok := ast.ElementType(collectionType)
			if !ok {
				return nil, typeError(expr.Collection.Pos(), "cannot slice %s", collectionType)
			}
			typ = &ast.SliceType{Elem: elemType}
		}
		c.setExpr(expr, typ, c.exprOrigin(expr.Collection))
		return typ, nil
	default:
		return nil, typeError(expr.Pos(), "unsupported expression %T", expr)
	}
}

func (c *checker) checkCall(locals *scope, expr *ast.CallExpr, allowPrint bool) (ast.Type, error) {
	argTypes := make([]ast.Type, 0, len(expr.Args))
	for _, arg := range expr.Args {
		argType, err := c.checkExpr(locals, arg, false)
		if err != nil {
			return nil, err
		}
		argTypes = append(argTypes, argType)
	}

	if expr.Callee == "print" {
		if !allowPrint {
			return nil, typeError(expr.Start, "print can only be used as a statement")
		}
		if len(argTypes) == 0 {
			return nil, typeError(expr.Start, "print expects at least 1 argument, got 0")
		}
		for i, argType := range argTypes {
			if !printableType(argType) {
				return nil, typeError(expr.Args[i].Pos(), "print does not support %s", argType)
			}
		}
		c.info.PrintCalls[expr] = argTypes
		c.setExpr(expr, argTypes[len(argTypes)-1], OriginUnknown)
		return argTypes[len(argTypes)-1], nil
	}
	if expr.Callee == "len" {
		if len(argTypes) != 1 {
			return nil, typeError(expr.Start, "len expects 1 argument, got %d", len(argTypes))
		}
		if !hasLength(argTypes[0]) {
			return nil, typeError(expr.Args[0].Pos(), "len does not support %s", argTypes[0])
		}
		c.info.LenCalls[expr] = argTypes[0]
		c.setExpr(expr, ast.IntType, OriginOwned)
		return ast.IntType, nil
	}
	if expr.Callee == "clone" {
		if len(argTypes) != 1 {
			return nil, typeError(expr.Start, "clone expects 1 argument, got %d", len(argTypes))
		}
		if !cloneableType(argTypes[0]) {
			return nil, typeError(expr.Args[0].Pos(), "clone does not support %s", argTypes[0])
		}
		c.info.CloneCalls[expr] = CloneSig{Type: argTypes[0]}
		c.setExpr(expr, argTypes[0], OriginFrameOwned)
		return argTypes[0], nil
	}
	if expr.Callee == "append" {
		if !allowPrint {
			return nil, typeError(expr.Start, "append can only be used as a statement")
		}
		if len(argTypes) != 2 {
			return nil, typeError(expr.Start, "append expects 2 arguments, got %d", len(argTypes))
		}
		listType, ok := argTypes[0].(*ast.ListType)
		if !ok {
			return nil, typeError(expr.Args[0].Pos(), "append expects list as first argument, got %s", argTypes[0])
		}
		if !ast.TypeEqual(argTypes[1], listType.Elem) {
			return nil, typeError(expr.Args[1].Pos(), "append value has type %s, want %s", argTypes[1], listType.Elem)
		}
		if err := c.checkCanMutate(expr.Args[0]); err != nil {
			return nil, err
		}
		c.info.AppendCalls[expr] = AppendSig{ListType: argTypes[0], ElemType: listType.Elem}
		c.setExpr(expr, argTypes[0], c.exprOrigin(expr.Args[0]))
		return argTypes[0], nil
	}

	sig, ok := c.info.Funcs[expr.Callee]
	if !ok {
		return nil, typeError(expr.Start, "undefined function %q", expr.Callee)
	}
	if len(argTypes) != len(sig.Params) {
		return nil, typeError(expr.Start, "%s expects %d arguments, got %d", expr.Callee, len(sig.Params), len(argTypes))
	}
	for i, argType := range argTypes {
		want := sig.Params[i].Type
		if !ast.TypeEqual(argType, want) {
			return nil, typeError(expr.Args[i].Pos(), "argument %d to %s has type %s, want %s", i+1, expr.Callee, argType, want)
		}
	}

	c.info.ResolvedCalls[expr] = sig
	c.setExpr(expr, sig.ReturnType, originForCallResult(sig.ReturnType))
	return sig.ReturnType, nil
}

func binaryType(pos token.Position, operator string, leftType ast.Type, rightType ast.Type) (ast.Type, error) {
	switch operator {
	case "+":
		if ast.TypeEqual(leftType, ast.StringType) && ast.TypeEqual(rightType, ast.StringType) {
			return ast.StringType, nil
		}
		if ast.TypeEqual(leftType, ast.IntType) && ast.TypeEqual(rightType, ast.IntType) {
			return ast.IntType, nil
		}
		if ast.TypeEqual(leftType, ast.FloatType) && ast.TypeEqual(rightType, ast.FloatType) {
			return ast.FloatType, nil
		}
		return nil, typeError(pos, "operator %q requires matching numeric operands or string operands, got %s and %s", operator, leftType, rightType)
	case "-", "*", "/":
		if ast.TypeEqual(leftType, ast.IntType) && ast.TypeEqual(rightType, ast.IntType) {
			return ast.IntType, nil
		}
		if ast.TypeEqual(leftType, ast.FloatType) && ast.TypeEqual(rightType, ast.FloatType) {
			return ast.FloatType, nil
		}
		return nil, typeError(pos, "operator %q requires matching numeric operands, got %s and %s", operator, leftType, rightType)
	case "==", "!=":
		if ast.TypeEqual(leftType, rightType) && comparableType(leftType) {
			return ast.BoolType, nil
		}
		return nil, typeError(pos, "operator %q requires matching comparable operands, got %s and %s", operator, leftType, rightType)
	case "<", "<=", ">", ">=":
		if ast.TypeEqual(leftType, rightType) && (ast.TypeEqual(leftType, ast.IntType) || ast.TypeEqual(leftType, ast.FloatType)) {
			return ast.BoolType, nil
		}
		return nil, typeError(pos, "operator %q requires matching numeric operands, got %s and %s", operator, leftType, rightType)
	case "in":
		if ast.TypeEqual(leftType, ast.StringType) && ast.TypeEqual(rightType, ast.StringType) {
			return ast.BoolType, nil
		}
		return nil, typeError(pos, "operator %q requires string operands, got %s and %s", operator, leftType, rightType)
	default:
		return nil, typeError(pos, "unsupported operator %q", operator)
	}
}

func comparableType(typ ast.Type) bool {
	return printableType(typ)
}

func printableType(typ ast.Type) bool {
	return ast.TypeEqual(typ, ast.IntType) ||
		ast.TypeEqual(typ, ast.FloatType) ||
		ast.TypeEqual(typ, ast.StringType) ||
		ast.TypeEqual(typ, ast.BoolType)
}

func hasLength(typ ast.Type) bool {
	if ast.TypeEqual(typ, ast.StringType) {
		return true
	}
	_, ok := ast.ElementType(typ)
	return ok
}

func isMutableCollection(typ ast.Type) bool {
	_, ok := ast.ElementType(typ)
	return ok
}

func cloneableType(typ ast.Type) bool {
	return ast.TypeEqual(typ, ast.StringType) || isMutableCollection(typ)
}

func dynamicType(typ ast.Type) bool {
	return ast.TypeEqual(typ, ast.StringType) || isMutableCollection(typ)
}

func originForParam(typ ast.Type) Origin {
	if dynamicType(typ) {
		return OriginBorrowed
	}
	return OriginOwned
}

func originForCallResult(typ ast.Type) Origin {
	if dynamicType(typ) {
		return OriginUnknown
	}
	return OriginOwned
}

func binaryOrigin(operator string, typ ast.Type) Origin {
	if operator == "+" && ast.TypeEqual(typ, ast.StringType) {
		return OriginScratch
	}
	return OriginOwned
}

func indexOrigin(collectionType ast.Type, collectionOrigin Origin) Origin {
	if ast.TypeEqual(collectionType, ast.StringType) {
		return collectionOrigin
	}
	elemType, ok := ast.ElementType(collectionType)
	if ok && dynamicType(elemType) {
		return collectionOrigin
	}
	return OriginOwned
}

func (c *checker) setExpr(expr ast.Expression, typ ast.Type, origin Origin) {
	c.info.ExprTypes[expr] = typ
	c.info.ExprOrigins[expr] = origin
}

func (c *checker) exprOrigin(expr ast.Expression) Origin {
	origin, ok := c.info.ExprOrigins[expr]
	if !ok {
		return OriginUnknown
	}
	return origin
}

func (c *checker) checkCanMutate(expr ast.Expression) error {
	switch c.exprOrigin(expr) {
	case OriginBorrowed:
		return typeError(expr.Pos(), "cannot mutate parameter-owned collection")
	case OriginUnknown:
		return typeError(expr.Pos(), "cannot mutate collection with unknown ownership")
	default:
		return nil
	}
}

func validateType(pos token.Position, typ ast.Type) error {
	switch typ := typ.(type) {
	case ast.ScalarType:
		return nil
	case *ast.ArrayType:
		if typ.Length <= 0 {
			return typeError(pos, "array length must be positive")
		}
		return validateType(pos, typ.Elem)
	case *ast.SliceType:
		return validateType(pos, typ.Elem)
	case *ast.ListType:
		return validateType(pos, typ.Elem)
	default:
		return typeError(pos, "unsupported type %s", typ)
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
