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

	fmt.Fprintln(&out, "#include <stdint.h>")
	fmt.Fprintln(&out)
	fmt.Fprint(&out, runtimec.Source)
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
	fmt.Fprintln(&out, "    return (int)trux_main();")
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
	for _, stmt := range fn.Body {
		if err := emitStmt(out, stmt); err != nil {
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
	for i, param := range fn.Params {
		if i > 0 {
			fmt.Fprint(out, ", ")
		}
		paramType, err := emitType(param.Type)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s %s", paramType, param.Name)
	}
	if len(fn.Params) == 0 {
		fmt.Fprint(out, "void")
	}
	fmt.Fprint(out, ")")
	if prototype {
		fmt.Fprintln(out, ";")
	}

	return nil
}

func emitStmt(out *bytes.Buffer, stmt ir.Stmt) error {
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
		fmt.Fprintf(out, "    %s %s = %s;\n", typ, stmt.Name, value)
	case *ir.ReturnStmt:
		value, err := emitExpr(stmt.Value)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "    return %s;\n", value)
	case *ir.PrintStmt:
		if stmt.Type != ast.IntType {
			return fmt.Errorf("unsupported print type %s", stmt.Type)
		}
		arg, err := emitExpr(stmt.Arg)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "    rt_print_int(%s);\n", arg)
	case *ir.ExprStmt:
		expr, err := emitExpr(stmt.Expr)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "    %s;\n", expr)
	default:
		return fmt.Errorf("unsupported IR statement %T", stmt)
	}

	return nil
}

func emitExpr(expr ir.Expr) (string, error) {
	switch expr := expr.(type) {
	case *ir.IdentExpr:
		return expr.Name, nil
	case *ir.IntLiteral:
		return expr.Value, nil
	case *ir.CallExpr:
		args := make([]string, 0, len(expr.Args))
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
		return fmt.Sprintf("(%s %s %s)", left, expr.Operator, right), nil
	default:
		return "", fmt.Errorf("unsupported IR expression %T", expr)
	}
}

func emitType(typ ast.Type) (string, error) {
	switch typ {
	case ast.IntType:
		return "int64_t", nil
	default:
		return "", fmt.Errorf("unsupported type %s", typ)
	}
}

func mangleFunc(name string) string {
	return "trux_" + name
}
