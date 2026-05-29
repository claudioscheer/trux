package c

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/claudioscheer/trux/internal/ast"
	"github.com/claudioscheer/trux/internal/ir"
	runtimec "github.com/claudioscheer/trux/internal/runtime/c"
)

func Generate(program *ir.Program) (string, error) {
	var out bytes.Buffer

	out.WriteString(runtimec.Source)
	fmt.Fprintln(&out)

	for _, fn := range program.Functions {
		if err := emitPrototype(&out, fn); err != nil {
			return "", err
		}
	}
	fmt.Fprintln(&out)

	for _, fn := range program.Functions {
		if err := emitFunc(&out, fn); err != nil {
			return "", err
		}
		fmt.Fprintln(&out)
	}

	fmt.Fprintln(&out, "int main(void) {")
	fmt.Fprintln(&out, "    rt_arena trux_arena;")
	fmt.Fprintln(&out, "    rt_arena_init(&trux_arena);")
	fmt.Fprintln(&out, "    int64_t trux_exit_code = trux_main(&trux_arena);")
	fmt.Fprintln(&out, "    rt_arena_deinit(&trux_arena);")
	fmt.Fprintln(&out, "    return (int)trux_exit_code;")
	fmt.Fprintln(&out, "}")

	return out.String(), nil
}

func emitPrototype(out *bytes.Buffer, fn *ir.Func) error {
	return emitSignature(out, fn, true)
}

func emitFunc(out *bytes.Buffer, fn *ir.Func) error {
	if err := emitSignature(out, fn, false); err != nil {
		return err
	}
	fmt.Fprintln(out, " {")
	if !funcUsesArena(fn) {
		fmt.Fprintln(out, "    (void)trux_arena;")
	}
	for _, stmt := range fn.Body {
		if err := emitStmt(out, stmt, 1); err != nil {
			return err
		}
	}
	fmt.Fprintln(out, "}")
	return nil
}

func emitSignature(out *bytes.Buffer, fn *ir.Func, prototype bool) error {
	cType, err := emitType(fn.ReturnType)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "%s %s(", cType, mangleFunc(fn.Name))
	fmt.Fprint(out, "rt_arena* trux_arena")
	for _, param := range fn.Params {
		fmt.Fprint(out, ", ")
		paramType, err := emitType(param.Type)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s %s", paramType, mangleIdent(param.Name))
	}
	fmt.Fprint(out, ")")
	if prototype {
		fmt.Fprintln(out, ";")
	}

	return nil
}

func emitStmt(out *bytes.Buffer, stmt ir.Stmt, level int) error {
	indent := strings.Repeat("    ", level)
	switch stmt := stmt.(type) {
	case *ir.LetStmt:
		typ, err := emitType(stmt.Type)
		if err != nil {
			return err
		}
		value, err := emitExpr(stmt.Value)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s%s %s = %s;\n", indent, typ, mangleIdent(stmt.Name), value)
	case *ir.ReturnStmt:
		value, err := emitExpr(stmt.Value)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%sreturn %s;\n", indent, value)
	case *ir.AssignStmt:
		value, err := emitExpr(stmt.Value)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s%s = %s;\n", indent, mangleIdent(stmt.Name), value)
	case *ir.IndexAssignStmt:
		collection, err := emitExpr(stmt.Target.Collection)
		if err != nil {
			return err
		}
		index, err := emitExpr(stmt.Target.Index)
		if err != nil {
			return err
		}
		value, err := emitExpr(stmt.Value)
		if err != nil {
			return err
		}
		setter, err := collectionHelper("set", stmt.Target.Collection.Type())
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s%s(%s, %s, %s);\n", indent, setter, collection, index, value)
	case *ir.IfStmt:
		condition, err := emitCondition(stmt.Condition)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%sif (%s) {\n", indent, condition)
		if err := emitStmts(out, stmt.Then, level+1); err != nil {
			return err
		}
		if len(stmt.Else) == 0 {
			fmt.Fprintf(out, "%s}\n", indent)
			break
		}
		fmt.Fprintf(out, "%s} else {\n", indent)
		if err := emitStmts(out, stmt.Else, level+1); err != nil {
			return err
		}
		fmt.Fprintf(out, "%s}\n", indent)
	case *ir.WhileStmt:
		condition, err := emitCondition(stmt.Condition)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%swhile (%s) {\n", indent, condition)
		if err := emitStmts(out, stmt.Body, level+1); err != nil {
			return err
		}
		fmt.Fprintf(out, "%s}\n", indent)
	case *ir.PrintStmt:
		if len(stmt.Args) != len(stmt.Types) {
			return fmt.Errorf("print arg/type count mismatch")
		}
		for i, argExpr := range stmt.Args {
			arg, err := emitExpr(argExpr)
			if err != nil {
				return err
			}
			switch {
			case ast.TypeEqual(stmt.Types[i], ast.IntType):
				fmt.Fprintf(out, "%srt_print_int(%s);\n", indent, arg)
			case ast.TypeEqual(stmt.Types[i], ast.FloatType):
				fmt.Fprintf(out, "%srt_print_float(%s);\n", indent, arg)
			case ast.TypeEqual(stmt.Types[i], ast.StringType):
				fmt.Fprintf(out, "%srt_print_string(%s);\n", indent, arg)
			case ast.TypeEqual(stmt.Types[i], ast.BoolType):
				fmt.Fprintf(out, "%srt_print_bool(%s);\n", indent, arg)
			default:
				return fmt.Errorf("unsupported print type %s", stmt.Types[i])
			}
		}
		fmt.Fprintf(out, "%srt_print_newline();\n", indent)
	case *ir.AppendStmt:
		list, err := emitExpr(stmt.List)
		if err != nil {
			return err
		}
		value, err := emitExpr(stmt.Value)
		if err != nil {
			return err
		}
		appendFn, err := listHelper("append", stmt.ListType)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s%s(%s, %s);\n", indent, appendFn, list, value)
	case *ir.ExprStmt:
		expr, err := emitExpr(stmt.Expr)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s%s;\n", indent, expr)
	default:
		return fmt.Errorf("unsupported IR statement %T", stmt)
	}

	return nil
}

func emitStmts(out *bytes.Buffer, stmts []ir.Stmt, level int) error {
	for _, stmt := range stmts {
		if err := emitStmt(out, stmt, level); err != nil {
			return err
		}
	}

	return nil
}

func funcUsesArena(fn *ir.Func) bool {
	return stmtsUseArena(fn.Body)
}

func stmtsUseArena(stmts []ir.Stmt) bool {
	for _, stmt := range stmts {
		if stmtUsesArena(stmt) {
			return true
		}
	}
	return false
}

func stmtUsesArena(stmt ir.Stmt) bool {
	switch stmt := stmt.(type) {
	case *ir.LetStmt:
		return exprUsesArena(stmt.Value)
	case *ir.ReturnStmt:
		return exprUsesArena(stmt.Value)
	case *ir.AssignStmt:
		return exprUsesArena(stmt.Value)
	case *ir.IndexAssignStmt:
		return exprUsesArena(stmt.Target) || exprUsesArena(stmt.Value)
	case *ir.IfStmt:
		return exprUsesArena(stmt.Condition) || stmtsUseArena(stmt.Then) || stmtsUseArena(stmt.Else)
	case *ir.WhileStmt:
		return exprUsesArena(stmt.Condition) || stmtsUseArena(stmt.Body)
	case *ir.PrintStmt:
		return exprsUseArena(stmt.Args)
	case *ir.AppendStmt:
		return exprUsesArena(stmt.List) || exprUsesArena(stmt.Value)
	case *ir.ExprStmt:
		return exprUsesArena(stmt.Expr)
	default:
		return false
	}
}

func exprsUseArena(exprs []ir.Expr) bool {
	for _, expr := range exprs {
		if exprUsesArena(expr) {
			return true
		}
	}
	return false
}

func exprUsesArena(expr ir.Expr) bool {
	switch expr := expr.(type) {
	case *ir.ArrayLiteral, *ir.ListLiteral, *ir.MakeExpr, *ir.CallExpr:
		return true
	case *ir.BinaryExpr:
		return exprUsesArena(expr.Left) || exprUsesArena(expr.Right) ||
			(expr.Operator == "+" && ast.TypeEqual(expr.Left.Type(), ast.StringType))
	case *ir.LenExpr:
		return exprUsesArena(expr.Value)
	case *ir.IndexExpr:
		return exprUsesArena(expr.Collection) || exprUsesArena(expr.Index)
	case *ir.SliceExpr:
		return exprUsesArena(expr.Collection) || exprUsesArena(expr.Start) || exprUsesArena(expr.End)
	default:
		return false
	}
}

func emitCondition(expr ir.Expr) (string, error) {
	condition, err := emitExpr(expr)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(condition, "(") && strings.HasSuffix(condition, ")") {
		return condition[1 : len(condition)-1], nil
	}
	return condition, nil
}

func emitExpr(expr ir.Expr) (string, error) {
	switch expr := expr.(type) {
	case *ir.IdentExpr:
		return mangleIdent(expr.Name), nil
	case *ir.IntLiteral:
		return expr.Value, nil
	case *ir.FloatLiteral:
		return expr.Value, nil
	case *ir.StringLiteral:
		return fmt.Sprintf("(rt_string){(const uint8_t*)%s, %d}", cStringLiteral(expr.Value), len(expr.Value)), nil
	case *ir.BoolLiteral:
		if expr.Value {
			return "true", nil
		}
		return "false", nil
	case *ir.ArrayLiteral:
		arrayType, ok := expr.Type().(*ast.ArrayType)
		if !ok {
			return "", fmt.Errorf("array literal has non-array type %s", expr.Type())
		}
		values, err := emitValueArray(arrayType.Elem, expr.Elements)
		if err != nil {
			return "", err
		}
		ctor, err := collectionHelper("from_values", expr.Type())
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s(trux_arena, %s, %d)", ctor, values, len(expr.Elements)), nil
	case *ir.ListLiteral:
		listType, ok := expr.Type().(*ast.ListType)
		if !ok {
			return "", fmt.Errorf("list literal has non-list type %s", expr.Type())
		}
		values, err := emitValueArray(listType.Elem, expr.Elements)
		if err != nil {
			return "", err
		}
		ctor, err := listHelper("from_values", expr.Type())
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s(trux_arena, %s, %d)", ctor, values, len(expr.Elements)), nil
	case *ir.MakeExpr:
		length, err := emitExpr(expr.Len)
		if err != nil {
			return "", err
		}
		makeFn, err := sliceMakeHelper(expr.Type())
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s(trux_arena, %s)", makeFn, length), nil
	case *ir.CallExpr:
		args := make([]string, 0, len(expr.Args)+1)
		args = append(args, "trux_arena")
		for _, arg := range expr.Args {
			emitArg, err := emitExpr(arg)
			if err != nil {
				return "", err
			}
			args = append(args, emitArg)
		}
		return fmt.Sprintf("%s(%s)", mangleFunc(expr.Callee), strings.Join(args, ", ")), nil
	case *ir.BinaryExpr:
		left, err := emitExpr(expr.Left)
		if err != nil {
			return "", err
		}
		right, err := emitExpr(expr.Right)
		if err != nil {
			return "", err
		}
		if expr.Operator == "in" {
			return fmt.Sprintf("rt_string_contains(%s, %s)", left, right), nil
		}
		if expr.Operator == "+" && ast.TypeEqual(expr.Left.Type(), ast.StringType) {
			return fmt.Sprintf("rt_string_concat(trux_arena, %s, %s)", left, right), nil
		}
		if ast.TypeEqual(expr.Left.Type(), ast.StringType) && (expr.Operator == "==" || expr.Operator == "!=") {
			equal := fmt.Sprintf("rt_string_equal(%s, %s)", left, right)
			if expr.Operator == "!=" {
				return fmt.Sprintf("!%s", equal), nil
			}
			return equal, nil
		}
		return fmt.Sprintf("(%s %s %s)", left, expr.Operator, right), nil
	case *ir.LenExpr:
		value, err := emitExpr(expr.Value)
		if err != nil {
			return "", err
		}
		if ast.TypeEqual(expr.Value.Type(), ast.StringType) {
			return fmt.Sprintf("((int64_t)%s.len)", value), nil
		}
		if _, ok := expr.Value.Type().(*ast.ListType); ok {
			return fmt.Sprintf("((int64_t)%s->len)", value), nil
		}
		return fmt.Sprintf("((int64_t)%s.len)", value), nil
	case *ir.IndexExpr:
		collection, err := emitExpr(expr.Collection)
		if err != nil {
			return "", err
		}
		index, err := emitExpr(expr.Index)
		if err != nil {
			return "", err
		}
		if ast.TypeEqual(expr.Collection.Type(), ast.StringType) {
			return fmt.Sprintf("rt_string_index(%s, %s)", collection, index), nil
		}
		getter, err := collectionHelper("get", expr.Collection.Type())
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s(%s, %s)", getter, collection, index), nil
	case *ir.SliceExpr:
		collection, err := emitExpr(expr.Collection)
		if err != nil {
			return "", err
		}
		hasStart, start, err := emitOptionalBound(expr.Start)
		if err != nil {
			return "", err
		}
		hasEnd, end, err := emitOptionalBound(expr.End)
		if err != nil {
			return "", err
		}
		if ast.TypeEqual(expr.Collection.Type(), ast.StringType) {
			return fmt.Sprintf("rt_string_slice(%s, %s, %s, %s, %s)", collection, hasStart, start, hasEnd, end), nil
		}
		sliceFn, err := collectionHelper("slice", expr.Collection.Type())
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s(%s, %s, %s, %s, %s)", sliceFn, collection, hasStart, start, hasEnd, end), nil
	default:
		return "", fmt.Errorf("unsupported IR expression %T", expr)
	}
}

func emitType(typ ast.Type) (string, error) {
	switch typ := typ.(type) {
	case ast.ScalarType:
		switch {
		case ast.TypeEqual(typ, ast.IntType):
			return "int64_t", nil
		case ast.TypeEqual(typ, ast.FloatType):
			return "double", nil
		case ast.TypeEqual(typ, ast.StringType):
			return "rt_string", nil
		case ast.TypeEqual(typ, ast.BoolType):
			return "bool", nil
		default:
			return "", fmt.Errorf("unsupported type %s", typ)
		}
	case *ast.ArrayType:
		name, err := elemRuntimeName(typ.Elem)
		if err != nil {
			return "", err
		}
		return "rt_array_" + name, nil
	case *ast.SliceType:
		name, err := elemRuntimeName(typ.Elem)
		if err != nil {
			return "", err
		}
		return "rt_slice_" + name, nil
	case *ast.ListType:
		name, err := elemRuntimeName(typ.Elem)
		if err != nil {
			return "", err
		}
		return "rt_list_" + name + "*", nil
	default:
		return "", fmt.Errorf("unsupported type %s", typ)
	}
}

func emitScalarType(typ ast.Type) (string, error) {
	switch {
	case ast.TypeEqual(typ, ast.IntType):
		return "int64_t", nil
	case ast.TypeEqual(typ, ast.FloatType):
		return "double", nil
	case ast.TypeEqual(typ, ast.StringType):
		return "rt_string", nil
	case ast.TypeEqual(typ, ast.BoolType):
		return "bool", nil
	default:
		return "", fmt.Errorf("unsupported type %s", typ)
	}
}

func emitValueArray(elemType ast.Type, elements []ir.Expr) (string, error) {
	if len(elements) == 0 {
		return "NULL", nil
	}
	cType, err := emitScalarType(elemType)
	if err != nil {
		return "", err
	}
	values := make([]string, 0, len(elements))
	for _, elem := range elements {
		value, err := emitExpr(elem)
		if err != nil {
			return "", err
		}
		values = append(values, value)
	}
	return fmt.Sprintf("(%s[]){%s}", cType, strings.Join(values, ", ")), nil
}

func emitOptionalBound(expr ir.Expr) (string, string, error) {
	if expr == nil {
		return "false", "0", nil
	}
	value, err := emitExpr(expr)
	if err != nil {
		return "", "", err
	}
	return "true", value, nil
}

func collectionHelper(op string, typ ast.Type) (string, error) {
	switch typ := typ.(type) {
	case *ast.ArrayType:
		name, err := elemRuntimeName(typ.Elem)
		if err != nil {
			return "", err
		}
		return "rt_array_" + name + "_" + op, nil
	case *ast.SliceType:
		name, err := elemRuntimeName(typ.Elem)
		if err != nil {
			return "", err
		}
		return "rt_slice_" + name + "_" + op, nil
	case *ast.ListType:
		return listHelper(op, typ)
	default:
		return "", fmt.Errorf("unsupported collection type %s", typ)
	}
}

func listHelper(op string, typ ast.Type) (string, error) {
	listType, ok := typ.(*ast.ListType)
	if !ok {
		return "", fmt.Errorf("unsupported list type %s", typ)
	}
	name, err := elemRuntimeName(listType.Elem)
	if err != nil {
		return "", err
	}
	return "rt_list_" + name + "_" + op, nil
}

func sliceMakeHelper(typ ast.Type) (string, error) {
	sliceType, ok := typ.(*ast.SliceType)
	if !ok {
		return "", fmt.Errorf("make expects slice type, got %s", typ)
	}
	name, err := elemRuntimeName(sliceType.Elem)
	if err != nil {
		return "", err
	}
	return "rt_make_slice_" + name, nil
}

func elemRuntimeName(typ ast.Type) (string, error) {
	switch {
	case ast.TypeEqual(typ, ast.IntType):
		return "int", nil
	case ast.TypeEqual(typ, ast.FloatType):
		return "float", nil
	case ast.TypeEqual(typ, ast.BoolType):
		return "bool", nil
	case ast.TypeEqual(typ, ast.StringType):
		return "string", nil
	default:
		return "", fmt.Errorf("unsupported element type %s", typ)
	}
}

func mangleFunc(name string) string {
	return "trux_" + name
}

func mangleIdent(name string) string {
	return fmt.Sprintf("trux_v_%d_%s", len(name), name)
}

func cStringLiteral(value string) string {
	var out strings.Builder
	out.WriteByte('"')
	for i := 0; i < len(value); i++ {
		ch := value[i]
		switch ch {
		case '"':
			out.WriteString("\\\"")
		case '\\':
			out.WriteString("\\\\")
		case '\n':
			out.WriteString("\\n")
		case '\t':
			out.WriteString("\\t")
		default:
			if ch >= 32 && ch <= 126 {
				out.WriteByte(ch)
			} else {
				fmt.Fprintf(&out, "\\%03o", ch)
			}
		}
	}
	out.WriteByte('"')
	return out.String()
}
