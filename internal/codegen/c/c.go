package c

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/claudioscheer/trux/internal/ast"
	"github.com/claudioscheer/trux/internal/ir"
	runtimec "github.com/claudioscheer/trux/internal/runtime/c"
	semtypes "github.com/claudioscheer/trux/internal/types"
)

const (
	durableArena = "trux_ctx->arena"
	frameArena   = "&trux_frame"
	resultArena  = "trux_result_arena"
)

type funcUsage struct {
	ctx         bool
	resultArena bool
}

type collectionFamily struct {
	name    string
	cType   string
	cloneFn string
}

func (u *funcUsage) noteArena(arena string) {
	switch arena {
	case durableArena:
		u.ctx = true
	case resultArena:
		u.resultArena = true
	}
}

func Generate(program *ir.Program) (string, error) {
	var out bytes.Buffer

	out.WriteString(runtimec.Source)
	fmt.Fprintln(&out)

	families, err := nestedCollectionFamilies(program)
	if err != nil {
		return "", err
	}
	if len(families) > 0 {
		for _, family := range families {
			fmt.Fprintf(&out, "#define RT_CLONE_VALUE_%s(ARENA, VALUE) %s((ARENA), (VALUE))\n", family.name, family.cloneFn)
			fmt.Fprintf(&out, "RT_DEFINE_COLLECTIONS(%s, %s)\n", family.name, family.cType)
		}
		fmt.Fprintln(&out)
	}

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
	fmt.Fprintln(&out, "    rt_context trux_ctx = {&trux_arena};")
	fmt.Fprintln(&out, "    int64_t trux_exit_code = trux_main(&trux_ctx, &trux_arena);")
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
	returnType, err := emitType(fn.ReturnType)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, " {")

	var body bytes.Buffer
	usage := &funcUsage{}
	fmt.Fprintln(&body, "    rt_arena trux_frame;")
	fmt.Fprintln(&body, "    rt_arena_init(&trux_frame);")
	fmt.Fprintf(&body, "    %s trux_return_value;\n", returnType)
	for _, stmt := range fn.Body {
		if err := emitStmt(&body, stmt, 1, usage); err != nil {
			return err
		}
	}
	if !usage.ctx {
		fmt.Fprintln(out, "    (void)trux_ctx;")
	}
	if !usage.resultArena {
		fmt.Fprintln(out, "    (void)trux_result_arena;")
	}
	out.Write(body.Bytes())
	fmt.Fprintln(out, "trux_return:")
	fmt.Fprintln(out, "    rt_arena_deinit(&trux_frame);")
	fmt.Fprintln(out, "    return trux_return_value;")
	fmt.Fprintln(out, "}")
	return nil
}

func emitSignature(out *bytes.Buffer, fn *ir.Func, prototype bool) error {
	cType, err := emitType(fn.ReturnType)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "%s %s(", cType, mangleFunc(fn.Name))
	fmt.Fprint(out, "rt_context* trux_ctx, rt_arena* trux_result_arena")
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

func emitStmt(out *bytes.Buffer, stmt ir.Stmt, level int, usage *funcUsage) error {
	indent := strings.Repeat("    ", level)
	switch stmt := stmt.(type) {
	case *ir.LetStmt:
		typ, err := emitType(stmt.Type)
		if err != nil {
			return err
		}
		value, err := emitExpr(stmt.Value, frameArena, usage)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s%s %s = %s;\n", indent, typ, mangleIdent(stmt.Name), value)
	case *ir.ReturnStmt:
		targetArena := frameArena
		_, directClone := stmt.Value.(*ir.CloneExpr)
		if directClone {
			targetArena = resultArena
		} else if ast.TypeEqual(stmt.Value.Type(), ast.StringType) && !needsReturnCopyOut(stmt.Value) {
			targetArena = resultArena
		}
		value, err := emitExpr(stmt.Value, targetArena, usage)
		if err != nil {
			return err
		}
		if !directClone && needsReturnCopyOut(stmt.Value) {
			value, err = emitClone(stmt.Value.Type(), resultArena, value, usage)
			if err != nil {
				return err
			}
		}
		fmt.Fprintf(out, "%strux_return_value = %s;\n", indent, value)
		fmt.Fprintf(out, "%sgoto trux_return;\n", indent)
	case *ir.AssignStmt:
		value, err := emitExpr(stmt.Value, frameArena, usage)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s%s = %s;\n", indent, mangleIdent(stmt.Name), value)
	case *ir.IndexAssignStmt:
		collection, err := emitExpr(stmt.Target.Collection, frameArena, usage)
		if err != nil {
			return err
		}
		index, err := emitExpr(stmt.Target.Index, frameArena, usage)
		if err != nil {
			return err
		}
		valueArena := collectionValueArena(stmt.Target.Collection)
		value, err := emitExpr(stmt.Value, valueArena, usage)
		if err != nil {
			return err
		}
		setter, err := collectionHelper("set", stmt.Target.Collection.Type())
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s%s(%s, %s, %s);\n", indent, setter, collection, index, value)
	case *ir.IfStmt:
		condition, err := emitCondition(stmt.Condition, usage)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%sif (%s) {\n", indent, condition)
		if err := emitStmts(out, stmt.Then, level+1, usage); err != nil {
			return err
		}
		if len(stmt.Else) == 0 {
			fmt.Fprintf(out, "%s}\n", indent)
			break
		}
		fmt.Fprintf(out, "%s} else {\n", indent)
		if err := emitStmts(out, stmt.Else, level+1, usage); err != nil {
			return err
		}
		fmt.Fprintf(out, "%s}\n", indent)
	case *ir.WhileStmt:
		condition, err := emitCondition(stmt.Condition, usage)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%swhile (%s) {\n", indent, condition)
		if err := emitStmts(out, stmt.Body, level+1, usage); err != nil {
			return err
		}
		fmt.Fprintf(out, "%s}\n", indent)
	case *ir.PrintStmt:
		if len(stmt.Args) != len(stmt.Types) {
			return fmt.Errorf("print arg/type count mismatch")
		}
		for i, argExpr := range stmt.Args {
			arg, err := emitExpr(argExpr, frameArena, usage)
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
		list, err := emitExpr(stmt.List, frameArena, usage)
		if err != nil {
			return err
		}
		valueArena := collectionValueArena(stmt.List)
		value, err := emitExpr(stmt.Value, valueArena, usage)
		if err != nil {
			return err
		}
		appendFn, err := listHelper("append", stmt.ListType)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s%s(%s, %s);\n", indent, appendFn, list, value)
	case *ir.WriteFileStmt:
		path, err := emitExpr(stmt.Path, frameArena, usage)
		if err != nil {
			return err
		}
		contents, err := emitExpr(stmt.Contents, frameArena, usage)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%srt_write_file(%s, %s);\n", indent, path, contents)
	case *ir.WriteCSVStmt:
		path, err := emitExpr(stmt.Path, frameArena, usage)
		if err != nil {
			return err
		}
		cells, err := emitExpr(stmt.Cells, frameArena, usage)
		if err != nil {
			return err
		}
		columns, err := emitExpr(stmt.Columns, frameArena, usage)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%srt_write_csv(%s, %s, %s);\n", indent, path, cells, columns)
	case *ir.ExprStmt:
		expr, err := emitExpr(stmt.Expr, frameArena, usage)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s%s;\n", indent, expr)
	default:
		return fmt.Errorf("unsupported IR statement %T", stmt)
	}

	return nil
}

func emitStmts(out *bytes.Buffer, stmts []ir.Stmt, level int, usage *funcUsage) error {
	for _, stmt := range stmts {
		if err := emitStmt(out, stmt, level, usage); err != nil {
			return err
		}
	}

	return nil
}

func needsReturnCopyOut(expr ir.Expr) bool {
	if !isDynamicType(expr.Type()) {
		return false
	}
	switch expr.Ownership() {
	case semtypes.OriginFrameOwned, semtypes.OriginScratch, semtypes.OriginUnknown:
		return true
	default:
		return false
	}
}

func collectionValueArena(collection ir.Expr) string {
	if collection.Ownership() == semtypes.OriginOwned {
		return durableArena
	}
	return frameArena
}

func emitCondition(expr ir.Expr, usage *funcUsage) (string, error) {
	condition, err := emitExpr(expr, frameArena, usage)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(condition, "(") && strings.HasSuffix(condition, ")") {
		return condition[1 : len(condition)-1], nil
	}
	return condition, nil
}

func emitExpr(expr ir.Expr, targetArena string, usage *funcUsage) (string, error) {
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
		values, err := emitValueArray(arrayType.Elem, expr.Elements, targetArena, usage)
		if err != nil {
			return "", err
		}
		ctor, err := collectionHelper("from_values", expr.Type())
		if err != nil {
			return "", err
		}
		usage.noteArena(targetArena)
		return fmt.Sprintf("%s(%s, %s, %d)", ctor, targetArena, values, len(expr.Elements)), nil
	case *ir.ListLiteral:
		listType, ok := expr.Type().(*ast.ListType)
		if !ok {
			return "", fmt.Errorf("list literal has non-list type %s", expr.Type())
		}
		values, err := emitValueArray(listType.Elem, expr.Elements, targetArena, usage)
		if err != nil {
			return "", err
		}
		ctor, err := listHelper("from_values", expr.Type())
		if err != nil {
			return "", err
		}
		usage.noteArena(targetArena)
		return fmt.Sprintf("%s(%s, %s, %d)", ctor, targetArena, values, len(expr.Elements)), nil
	case *ir.MakeExpr:
		length, err := emitExpr(expr.Len, frameArena, usage)
		if err != nil {
			return "", err
		}
		makeFn, err := sliceMakeHelper(expr.Type())
		if err != nil {
			return "", err
		}
		usage.noteArena(targetArena)
		return fmt.Sprintf("%s(%s, %s)", makeFn, targetArena, length), nil
	case *ir.CloneExpr:
		value, err := emitExpr(expr.Value, frameArena, usage)
		if err != nil {
			return "", err
		}
		return emitClone(expr.Value.Type(), targetArena, value, usage)
	case *ir.ReadLineExpr:
		usage.noteArena(targetArena)
		return fmt.Sprintf("rt_read_line(%s)", targetArena), nil
	case *ir.ReadIntExpr:
		return "rt_read_int(&trux_frame)", nil
	case *ir.ReadFloatExpr:
		return "rt_read_float(&trux_frame)", nil
	case *ir.ReadBoolExpr:
		return "rt_read_bool(&trux_frame)", nil
	case *ir.ReadFileExpr:
		path, err := emitExpr(expr.Path, frameArena, usage)
		if err != nil {
			return "", err
		}
		usage.noteArena(targetArena)
		return fmt.Sprintf("rt_read_file(%s, %s)", targetArena, path), nil
	case *ir.ReadCSVExpr:
		path, err := emitExpr(expr.Path, frameArena, usage)
		if err != nil {
			return "", err
		}
		columns, err := emitExpr(expr.Columns, frameArena, usage)
		if err != nil {
			return "", err
		}
		usage.noteArena(targetArena)
		return fmt.Sprintf("rt_read_csv(%s, %s, %s)", targetArena, path, columns), nil
	case *ir.CallExpr:
		args := make([]string, 0, len(expr.Args)+1)
		callResultArena := frameArena
		if isDynamicType(expr.Type()) {
			callResultArena = targetArena
		}
		usage.ctx = true
		usage.noteArena(callResultArena)
		args = append(args, "trux_ctx", callResultArena)
		for _, arg := range expr.Args {
			argArena := frameArena
			if isDynamicType(expr.Type()) && ast.TypeEqual(arg.Type(), ast.StringType) {
				argArena = callResultArena
			}
			emitArg, err := emitExpr(arg, argArena, usage)
			if err != nil {
				return "", err
			}
			args = append(args, emitArg)
		}
		return fmt.Sprintf("%s(%s)", mangleFunc(expr.Callee), strings.Join(args, ", ")), nil
	case *ir.BinaryExpr:
		left, err := emitExpr(expr.Left, targetArena, usage)
		if err != nil {
			return "", err
		}
		right, err := emitExpr(expr.Right, targetArena, usage)
		if err != nil {
			return "", err
		}
		if expr.Operator == "in" {
			return fmt.Sprintf("rt_string_contains(%s, %s)", left, right), nil
		}
		if expr.Operator == "+" && ast.TypeEqual(expr.Left.Type(), ast.StringType) {
			usage.noteArena(targetArena)
			return fmt.Sprintf("rt_string_concat(%s, %s, %s)", targetArena, left, right), nil
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
		value, err := emitExpr(expr.Value, frameArena, usage)
		if err != nil {
			return "", err
		}
		if ast.TypeEqual(expr.Value.Type(), ast.StringType) {
			return fmt.Sprintf("rt_checked_len_i64(%s.len)", value), nil
		}
		if _, ok := expr.Value.Type().(*ast.ListType); ok {
			return fmt.Sprintf("rt_checked_len_i64(%s->len)", value), nil
		}
		return fmt.Sprintf("rt_checked_len_i64(%s.len)", value), nil
	case *ir.IndexExpr:
		collection, err := emitExpr(expr.Collection, targetArena, usage)
		if err != nil {
			return "", err
		}
		index, err := emitExpr(expr.Index, frameArena, usage)
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
		collection, err := emitExpr(expr.Collection, targetArena, usage)
		if err != nil {
			return "", err
		}
		hasStart, start, err := emitOptionalBound(expr.Start, usage)
		if err != nil {
			return "", err
		}
		hasEnd, end, err := emitOptionalBound(expr.End, usage)
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

func emitValueArray(elemType ast.Type, elements []ir.Expr, targetArena string, usage *funcUsage) (string, error) {
	if len(elements) == 0 {
		return "NULL", nil
	}
	cType, err := emitType(elemType)
	if err != nil {
		return "", err
	}
	values := make([]string, 0, len(elements))
	for _, elem := range elements {
		value, err := emitExpr(elem, targetArena, usage)
		if err != nil {
			return "", err
		}
		values = append(values, value)
	}
	return fmt.Sprintf("(%s[]){%s}", cType, strings.Join(values, ", ")), nil
}

func emitClone(typ ast.Type, targetArena string, value string, usage *funcUsage) (string, error) {
	cloneFn, err := cloneHelper(typ)
	if err != nil {
		return "", err
	}
	usage.noteArena(targetArena)
	return fmt.Sprintf("%s(%s, %s)", cloneFn, targetArena, value), nil
}

func emitOptionalBound(expr ir.Expr, usage *funcUsage) (string, string, error) {
	if expr == nil {
		return "false", "0", nil
	}
	value, err := emitExpr(expr, frameArena, usage)
	if err != nil {
		return "", "", err
	}
	return "true", value, nil
}

func isCollectionType(typ ast.Type) bool {
	_, ok := ast.ElementType(typ)
	return ok
}

func isDynamicType(typ ast.Type) bool {
	return ast.TypeEqual(typ, ast.StringType) || isCollectionType(typ)
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

func cloneHelper(typ ast.Type) (string, error) {
	if ast.TypeEqual(typ, ast.StringType) {
		return "rt_string_clone", nil
	}
	if isCollectionType(typ) {
		return collectionHelper("clone", typ)
	}
	return "", fmt.Errorf("clone does not support %s", typ)
}

func elemRuntimeName(typ ast.Type) (string, error) {
	switch typ := typ.(type) {
	case ast.ScalarType:
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
	case *ast.ArrayType:
		name, err := elemRuntimeName(typ.Elem)
		if err != nil {
			return "", err
		}
		return "array_" + name, nil
	case *ast.SliceType:
		name, err := elemRuntimeName(typ.Elem)
		if err != nil {
			return "", err
		}
		return "slice_" + name, nil
	case *ast.ListType:
		name, err := elemRuntimeName(typ.Elem)
		if err != nil {
			return "", err
		}
		return "list_" + name, nil
	default:
		return "", fmt.Errorf("unsupported element type %s", typ)
	}
}

func nestedCollectionFamilies(program *ir.Program) ([]collectionFamily, error) {
	seen := map[string]bool{}
	var families []collectionFamily
	for _, fn := range program.Functions {
		if err := collectNestedCollectionFamilies(fn.ReturnType, seen, &families); err != nil {
			return nil, err
		}
		for _, param := range fn.Params {
			if err := collectNestedCollectionFamilies(param.Type, seen, &families); err != nil {
				return nil, err
			}
		}
		for _, stmt := range fn.Body {
			if err := collectStmtNestedCollectionFamilies(stmt, seen, &families); err != nil {
				return nil, err
			}
		}
	}
	return families, nil
}

func collectStmtNestedCollectionFamilies(stmt ir.Stmt, seen map[string]bool, families *[]collectionFamily) error {
	switch stmt := stmt.(type) {
	case *ir.LetStmt:
		if err := collectNestedCollectionFamilies(stmt.Type, seen, families); err != nil {
			return err
		}
		return collectExprNestedCollectionFamilies(stmt.Value, seen, families)
	case *ir.ReturnStmt:
		return collectExprNestedCollectionFamilies(stmt.Value, seen, families)
	case *ir.AssignStmt:
		return collectExprNestedCollectionFamilies(stmt.Value, seen, families)
	case *ir.IndexAssignStmt:
		if err := collectExprNestedCollectionFamilies(stmt.Target, seen, families); err != nil {
			return err
		}
		return collectExprNestedCollectionFamilies(stmt.Value, seen, families)
	case *ir.IfStmt:
		if err := collectExprNestedCollectionFamilies(stmt.Condition, seen, families); err != nil {
			return err
		}
		for _, inner := range stmt.Then {
			if err := collectStmtNestedCollectionFamilies(inner, seen, families); err != nil {
				return err
			}
		}
		for _, inner := range stmt.Else {
			if err := collectStmtNestedCollectionFamilies(inner, seen, families); err != nil {
				return err
			}
		}
	case *ir.WhileStmt:
		if err := collectExprNestedCollectionFamilies(stmt.Condition, seen, families); err != nil {
			return err
		}
		for _, inner := range stmt.Body {
			if err := collectStmtNestedCollectionFamilies(inner, seen, families); err != nil {
				return err
			}
		}
	case *ir.PrintStmt:
		for i, arg := range stmt.Args {
			if err := collectNestedCollectionFamilies(stmt.Types[i], seen, families); err != nil {
				return err
			}
			if err := collectExprNestedCollectionFamilies(arg, seen, families); err != nil {
				return err
			}
		}
	case *ir.AppendStmt:
		if err := collectNestedCollectionFamilies(stmt.ListType, seen, families); err != nil {
			return err
		}
		if err := collectNestedCollectionFamilies(stmt.ElemType, seen, families); err != nil {
			return err
		}
		if err := collectExprNestedCollectionFamilies(stmt.List, seen, families); err != nil {
			return err
		}
		return collectExprNestedCollectionFamilies(stmt.Value, seen, families)
	case *ir.WriteFileStmt:
		if err := collectExprNestedCollectionFamilies(stmt.Path, seen, families); err != nil {
			return err
		}
		return collectExprNestedCollectionFamilies(stmt.Contents, seen, families)
	case *ir.WriteCSVStmt:
		if err := collectExprNestedCollectionFamilies(stmt.Path, seen, families); err != nil {
			return err
		}
		if err := collectExprNestedCollectionFamilies(stmt.Cells, seen, families); err != nil {
			return err
		}
		return collectExprNestedCollectionFamilies(stmt.Columns, seen, families)
	case *ir.ExprStmt:
		return collectExprNestedCollectionFamilies(stmt.Expr, seen, families)
	default:
		return fmt.Errorf("unsupported IR statement %T", stmt)
	}
	return nil
}

func collectExprNestedCollectionFamilies(expr ir.Expr, seen map[string]bool, families *[]collectionFamily) error {
	if expr == nil {
		return nil
	}
	if err := collectNestedCollectionFamilies(expr.Type(), seen, families); err != nil {
		return err
	}
	switch expr := expr.(type) {
	case *ir.ArrayLiteral:
		for _, elem := range expr.Elements {
			if err := collectExprNestedCollectionFamilies(elem, seen, families); err != nil {
				return err
			}
		}
	case *ir.ListLiteral:
		for _, elem := range expr.Elements {
			if err := collectExprNestedCollectionFamilies(elem, seen, families); err != nil {
				return err
			}
		}
	case *ir.MakeExpr:
		return collectExprNestedCollectionFamilies(expr.Len, seen, families)
	case *ir.CallExpr:
		for _, arg := range expr.Args {
			if err := collectExprNestedCollectionFamilies(arg, seen, families); err != nil {
				return err
			}
		}
	case *ir.CloneExpr:
		return collectExprNestedCollectionFamilies(expr.Value, seen, families)
	case *ir.BinaryExpr:
		if err := collectExprNestedCollectionFamilies(expr.Left, seen, families); err != nil {
			return err
		}
		return collectExprNestedCollectionFamilies(expr.Right, seen, families)
	case *ir.LenExpr:
		return collectExprNestedCollectionFamilies(expr.Value, seen, families)
	case *ir.IndexExpr:
		if err := collectExprNestedCollectionFamilies(expr.Collection, seen, families); err != nil {
			return err
		}
		return collectExprNestedCollectionFamilies(expr.Index, seen, families)
	case *ir.SliceExpr:
		if err := collectExprNestedCollectionFamilies(expr.Collection, seen, families); err != nil {
			return err
		}
		if err := collectExprNestedCollectionFamilies(expr.Start, seen, families); err != nil {
			return err
		}
		return collectExprNestedCollectionFamilies(expr.End, seen, families)
	}
	return nil
}

func collectNestedCollectionFamilies(typ ast.Type, seen map[string]bool, families *[]collectionFamily) error {
	elem, ok := ast.ElementType(typ)
	if !ok {
		return nil
	}
	if err := collectNestedCollectionFamilies(elem, seen, families); err != nil {
		return err
	}
	if !isCollectionType(elem) {
		return nil
	}

	name, err := elemRuntimeName(elem)
	if err != nil {
		return err
	}
	if seen[name] {
		return nil
	}
	cType, err := emitType(elem)
	if err != nil {
		return err
	}
	cloneFn, err := cloneHelper(elem)
	if err != nil {
		return err
	}
	seen[name] = true
	*families = append(*families, collectionFamily{name: name, cType: cType, cloneFn: cloneFn})
	return nil
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
