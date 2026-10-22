package c

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/claudioscheer/trux/internal/ast"
	"github.com/claudioscheer/trux/internal/ir"
	runtimec "github.com/claudioscheer/trux/internal/runtime/c"
	"github.com/claudioscheer/trux/internal/token"
	semtypes "github.com/claudioscheer/trux/internal/types"
)

const (
	durableArena = "trux_ctx->arena"
	frameArena   = "&trux_frame"
	resultArena  = "trux_result_arena"
)

var runtimeTuples = map[string]bool{
	"trux_tuple_string_error":      true,
	"trux_tuple_list_string_error": true,
	"trux_tuple_slice_int_error":   true,
}

type funcEmitter struct {
	program *ir.Program
	opts    Options
	fn      *ir.Func
	temp    int
	loops   []string
	loopID  int
	marks   int
	scratch string
}

func (e *funcEmitter) pushLoop() string {
	label := fmt.Sprintf("trux_continue_%d", e.loopID)
	e.loopID++
	e.loops = append(e.loops, label)
	return label
}

func (e *funcEmitter) popLoop() {
	e.loops = e.loops[:len(e.loops)-1]
}

func functionNeedsFrame(fn *ir.Func) bool {
	return stmtsNeedFrame(fn.Body)
}

func stmtsNeedFrame(stmts []ir.Stmt) bool {
	for _, stmt := range stmts {
		if stmt != nil && stmtNeedsFrame(stmt) {
			return true
		}
	}
	return false
}

func stmtNeedsFrame(stmt ir.Stmt) bool {
	switch stmt := stmt.(type) {
	case *ir.LetStmt:
		return exprNeedsFrame(stmt.Value)
	case *ir.ReturnStmt:
		return exprNeedsFrame(stmt.Value)
	case *ir.AssignStmt:
		return exprNeedsFrame(stmt.Value)
	case *ir.IndexAssignStmt, *ir.AppendStmt:
		return true
	case *ir.FieldAssignStmt:
		return exprNeedsFrame(stmt.Target) || exprNeedsFrame(stmt.Value)
	case *ir.DerefAssignStmt:
		return exprNeedsFrame(stmt.Target) || exprNeedsFrame(stmt.Value)
	case *ir.DeleteStmt:
		return exprNeedsFrame(stmt.Map) || exprNeedsFrame(stmt.Key)
	case *ir.IfStmt:
		return exprNeedsFrame(stmt.Condition) || stmtsNeedFrame(stmt.Then) || stmtsNeedFrame(stmt.Else)
	case *ir.ForStmt:
		return stmtNeedsFrame(stmt.Init) || exprNeedsFrame(stmt.Condition) || stmtsNeedFrame(stmt.Body) || stmtNeedsFrame(stmt.Post)
	case *ir.BreakStmt, *ir.ContinueStmt, *ir.GPUSyncStmt:
		return false
	case *ir.PrintStmt:
		for _, arg := range stmt.Args {
			if exprNeedsFrame(arg) {
				return true
			}
		}
		return false
	case *ir.AssertStmt:
		return exprNeedsFrame(stmt.Condition) || exprNeedsFrame(stmt.Message)
	case *ir.WriteFileStmt:
		return exprNeedsFrame(stmt.Path) || exprNeedsFrame(stmt.Contents)
	case *ir.WriteCSVStmt:
		return exprNeedsFrame(stmt.Path) || exprNeedsFrame(stmt.Cells) || exprNeedsFrame(stmt.Columns)
	case *ir.WritePPMStmt:
		return exprNeedsFrame(stmt.Path) || exprNeedsFrame(stmt.Pixels) || exprNeedsFrame(stmt.Width) || exprNeedsFrame(stmt.Height)
	case *ir.GPUCopyStmt:
		return exprNeedsFrame(stmt.First) || exprNeedsFrame(stmt.Second)
	case *ir.GPUFreeStmt:
		return exprNeedsFrame(stmt.Buffer)
	case *ir.GPULaunchStmt:
		return true
	case *ir.TimeSleepStmt:
		return exprNeedsFrame(stmt.Millis)
	case *ir.HTTPServeStmt, *ir.HTTPListenStmt, *ir.HTTPRouteStmt, *ir.HTTPUseStmt, *ir.HTTPNotFoundStmt:
		return true
	case *ir.HTTPRespondStmt:
		return exprNeedsFrame(stmt.Request) || exprNeedsFrame(stmt.Status) || exprNeedsFrame(stmt.ContentType) || exprNeedsFrame(stmt.Body)
	case *ir.HTTPTextStmt:
		return exprNeedsFrame(stmt.Request) || exprNeedsFrame(stmt.Status) || exprNeedsFrame(stmt.Body)
	case *ir.HTTPJSONStmt:
		return exprNeedsFrame(stmt.Request) || exprNeedsFrame(stmt.Status) || exprNeedsFrame(stmt.Body)
	case *ir.HTTPStatusStmt:
		return exprNeedsFrame(stmt.Request) || exprNeedsFrame(stmt.Status)
	case *ir.HTTPSetStmt:
		return exprNeedsFrame(stmt.Request) || exprNeedsFrame(stmt.Key) || exprNeedsFrame(stmt.Val)
	case *ir.HTTPSetHeaderStmt:
		return exprNeedsFrame(stmt.Request) || exprNeedsFrame(stmt.Name) || exprNeedsFrame(stmt.Val)
	case *ir.ExprStmt:
		return exprNeedsFrame(stmt.Expr)
	default:
		return true
	}
}

func exprNeedsFrame(expr ir.Expr) bool {
	if expr == nil {
		return false
	}
	switch expr := expr.(type) {
	case *ir.IdentExpr, *ir.IntLiteral, *ir.FloatLiteral, *ir.StringLiteral, *ir.BoolLiteral, *ir.NilLiteral, *ir.GPUCoordExpr, *ir.TimeNowExpr:
		return false
	case *ir.ReadIntExpr, *ir.ReadFloatExpr, *ir.ReadBoolExpr:
		return true
	case *ir.ImageDimensionExpr:
		return exprNeedsFrame(expr.Path)
	case *ir.PathQueryExpr:
		return exprNeedsFrame(expr.Path)
	case *ir.ArrayLiteral, *ir.ListLiteral, *ir.StructLiteral, *ir.MapLiteral, *ir.NewExpr, *ir.KeysExpr, *ir.ValuesExpr, *ir.MakeExpr, *ir.CloneExpr, *ir.ReadLineExpr, *ir.ReadFileExpr, *ir.ReadCSVExpr, *ir.ReadPPMExpr, *ir.HTTPRequestStringExpr, *ir.GPUAllocExpr, *ir.GPUFromExpr, *ir.CreateDirExpr, *ir.RemoveExpr, *ir.ReadDirExpr:
		return true
	case *ir.FieldExpr:
		return exprNeedsFrame(expr.Object)
	case *ir.MapLookupExpr:
		return exprNeedsFrame(expr.Map) || exprNeedsFrame(expr.Key)
	case *ir.CallExpr:
		if isDynamicType(expr.Type()) {
			return true
		}
		for _, arg := range expr.Args {
			if exprNeedsFrame(arg) {
				return true
			}
		}
		return false
	case *ir.SQLiteCallExpr:
		if isDynamicType(expr.Type()) {
			return true
		}
		for _, arg := range expr.Args {
			if exprNeedsFrame(arg) {
				return true
			}
		}
		return false
	case *ir.BinaryExpr:
		if expr.Operator == "+" && ast.TypeEqual(expr.Type(), ast.StringType) {
			return true
		}
		return exprNeedsFrame(expr.Left) || exprNeedsFrame(expr.Right)
	case *ir.UnaryExpr:
		if expr.Operator == "&" {
			return true
		}
		return exprNeedsFrame(expr.Operand)
	case *ir.LenExpr:
		return exprNeedsFrame(expr.Value)
	case *ir.IndexExpr:
		return exprNeedsFrame(expr.Collection) || exprNeedsFrame(expr.Index)
	case *ir.SliceExpr:
		return exprNeedsFrame(expr.Collection) || exprNeedsFrame(expr.Start) || exprNeedsFrame(expr.End)
	default:
		return true
	}
}

func loopHasContinue(stmts []ir.Stmt) bool {
	for _, stmt := range stmts {
		switch stmt := stmt.(type) {
		case *ir.ContinueStmt:
			return true
		case *ir.IfStmt:
			if loopHasContinue(stmt.Then) || loopHasContinue(stmt.Else) {
				return true
			}
		}
	}
	return false
}

// forLoopCanRewind reports whether memory allocated by this loop dies at the
// end of the iteration. A return copies its value out and jumps to the
// epilogue, so it does not by itself keep the loop from rewinding.
func forLoopCanRewind(stmt *ir.ForStmt) bool {
	if !forLoopAllocates(stmt) {
		return false
	}
	declared := map[string]bool{}
	tainted := map[string]bool{}
	if stmtsPublishFrame(stmt.Body, declared, tainted) {
		return false
	}
	if exprPublishesFrame(stmt.Condition, declared, tainted) {
		return false
	}
	if stmt.Post != nil && stmtPublishesFrame(stmt.Post, declared, tainted) {
		return false
	}
	return true
}

func forLoopAllocates(stmt *ir.ForStmt) bool {
	return exprNeedsFrame(stmt.Condition) || stmtsAllocateInLoop(stmt.Body) || stmtAllocatesInLoop(stmt.Post)
}

func stmtsAllocateInLoop(stmts []ir.Stmt) bool {
	for _, stmt := range stmts {
		if stmt != nil && stmtAllocatesInLoop(stmt) {
			return true
		}
	}
	return false
}

func stmtAllocatesInLoop(stmt ir.Stmt) bool {
	if stmt == nil {
		return false
	}
	switch stmt := stmt.(type) {
	case *ir.AppendStmt:
		return exprNeedsFrame(stmt.Value) || exprNeedsFrame(stmt.List)
	case *ir.IndexAssignStmt:
		return exprNeedsFrame(stmt.Value) || exprNeedsFrame(stmt.Target.Collection) || exprNeedsFrame(stmt.Target.Index)
	case *ir.ForStmt:
		return forLoopAllocates(stmt)
	default:
		return stmtNeedsFrame(stmt)
	}
}

func stmtsPublishFrame(stmts []ir.Stmt, declared map[string]bool, tainted map[string]bool) bool {
	for _, stmt := range stmts {
		if stmt != nil && stmtPublishesFrame(stmt, declared, tainted) {
			return true
		}
	}
	return false
}

func stmtPublishesFrame(stmt ir.Stmt, declared map[string]bool, tainted map[string]bool) bool {
	switch stmt := stmt.(type) {
	case *ir.LetStmt:
		if exprPublishesFrame(stmt.Value, declared, tainted) {
			return true
		}
		declared[stmt.Name] = true
		if valueReferencesFrame(stmt.Value, tainted) {
			tainted[stmt.Name] = true
		}
		return false
	case *ir.AssignStmt:
		if exprPublishesFrame(stmt.Value, declared, tainted) {
			return true
		}
		refs := valueReferencesFrame(stmt.Value, tainted)
		if !declared[stmt.Name] {
			return refs
		}
		if refs {
			tainted[stmt.Name] = true
		} else {
			delete(tainted, stmt.Name)
		}
		return false
	case *ir.AppendStmt:
		if exprPublishesFrame(stmt.List, declared, tainted) || exprPublishesFrame(stmt.Value, declared, tainted) {
			return true
		}
		return valueReferencesFrame(stmt.Value, tainted)
	case *ir.IndexAssignStmt:
		if exprPublishesFrame(stmt.Target.Collection, declared, tainted) || exprPublishesFrame(stmt.Target.Index, declared, tainted) || exprPublishesFrame(stmt.Value, declared, tainted) {
			return true
		}
		if _, isMap := stmt.Target.Collection.Type().(*ast.MapType); isMap && valueReferencesFrame(stmt.Target.Index, tainted) {
			return true
		}
		return valueReferencesFrame(stmt.Value, tainted)
	case *ir.FieldAssignStmt, *ir.DerefAssignStmt:
		return true
	case *ir.DeleteStmt:
		return exprPublishesFrame(stmt.Map, declared, tainted) || exprPublishesFrame(stmt.Key, declared, tainted)
	case *ir.IfStmt:
		if exprPublishesFrame(stmt.Condition, declared, tainted) {
			return true
		}
		thenDeclared := copyStringSet(declared)
		thenTainted := copyStringSet(tainted)
		if stmtsPublishFrame(stmt.Then, thenDeclared, thenTainted) {
			return true
		}
		elseDeclared := copyStringSet(declared)
		elseTainted := copyStringSet(tainted)
		if stmtsPublishFrame(stmt.Else, elseDeclared, elseTainted) {
			return true
		}
		for name := range declared {
			if thenTainted[name] || elseTainted[name] {
				tainted[name] = true
			} else {
				delete(tainted, name)
			}
		}
		return false
	case *ir.ForStmt:
		childDeclared := copyStringSet(declared)
		childTainted := copyStringSet(tainted)
		if init, ok := stmt.Init.(*ir.LetStmt); ok {
			if exprPublishesFrame(init.Value, childDeclared, childTainted) {
				return true
			}
			childDeclared[init.Name] = true
			if valueReferencesFrame(init.Value, childTainted) {
				childTainted[init.Name] = true
			}
		} else if stmt.Init != nil && stmtPublishesFrame(stmt.Init, childDeclared, childTainted) {
			return true
		}
		if stmtsPublishFrame(stmt.Body, childDeclared, childTainted) ||
			exprPublishesFrame(stmt.Condition, childDeclared, childTainted) ||
			(stmt.Post != nil && stmtPublishesFrame(stmt.Post, childDeclared, childTainted)) {
			return true
		}
		for name := range declared {
			if childTainted[name] {
				tainted[name] = true
			}
		}
		return false
	case *ir.ReturnStmt:
		return exprPublishesFrame(stmt.Value, declared, tainted)
	case *ir.PrintStmt:
		for _, arg := range stmt.Args {
			if exprPublishesFrame(arg, declared, tainted) {
				return true
			}
		}
		return false
	case *ir.AssertStmt:
		return exprPublishesFrame(stmt.Condition, declared, tainted) || exprPublishesFrame(stmt.Message, declared, tainted)
	case *ir.ExprStmt:
		return exprPublishesFrame(stmt.Expr, declared, tainted)
	case *ir.WriteFileStmt:
		return exprPublishesFrame(stmt.Path, declared, tainted) || exprPublishesFrame(stmt.Contents, declared, tainted)
	case *ir.WriteCSVStmt:
		return exprPublishesFrame(stmt.Path, declared, tainted) || exprPublishesFrame(stmt.Cells, declared, tainted) || exprPublishesFrame(stmt.Columns, declared, tainted)
	case *ir.WritePPMStmt:
		return exprPublishesFrame(stmt.Path, declared, tainted) || exprPublishesFrame(stmt.Pixels, declared, tainted) || exprPublishesFrame(stmt.Width, declared, tainted) || exprPublishesFrame(stmt.Height, declared, tainted)
	case *ir.TimeSleepStmt:
		return exprPublishesFrame(stmt.Millis, declared, tainted)
	case *ir.BreakStmt, *ir.ContinueStmt, *ir.GPUSyncStmt:
		return false
	case *ir.GPUCopyStmt, *ir.GPUFreeStmt, *ir.GPULaunchStmt, *ir.HTTPServeStmt, *ir.HTTPListenStmt, *ir.HTTPRouteStmt, *ir.HTTPUseStmt, *ir.HTTPNotFoundStmt, *ir.HTTPSetStmt, *ir.HTTPSetHeaderStmt, *ir.HTTPTextStmt, *ir.HTTPJSONStmt, *ir.HTTPStatusStmt, *ir.HTTPRespondStmt:
		return true
	default:
		return true
	}
}

func exprPublishesFrame(expr ir.Expr, declared map[string]bool, tainted map[string]bool) bool {
	if expr == nil {
		return false
	}
	switch expr := expr.(type) {
	case *ir.IdentExpr, *ir.IntLiteral, *ir.FloatLiteral, *ir.StringLiteral, *ir.BoolLiteral, *ir.NilLiteral, *ir.NewExpr, *ir.GPUCoordExpr, *ir.TimeNowExpr, *ir.ReadLineExpr, *ir.ReadIntExpr, *ir.ReadFloatExpr, *ir.ReadBoolExpr:
		return false
	case *ir.FieldExpr:
		return exprPublishesFrame(expr.Object, declared, tainted)
	case *ir.MapLookupExpr:
		return exprPublishesFrame(expr.Map, declared, tainted) || exprPublishesFrame(expr.Key, declared, tainted)
	case *ir.KeysExpr:
		return exprPublishesFrame(expr.Map, declared, tainted)
	case *ir.ValuesExpr:
		return exprPublishesFrame(expr.Map, declared, tainted)
	case *ir.StructLiteral:
		for _, f := range expr.Fields {
			if exprPublishesFrame(f.Value, declared, tainted) {
				return true
			}
		}
		return false
	case *ir.MapLiteral:
		for _, entry := range expr.Entries {
			if exprPublishesFrame(entry.Key, declared, tainted) || exprPublishesFrame(entry.Value, declared, tainted) {
				return true
			}
		}
		return false
	case *ir.CallExpr:
		for _, arg := range expr.Args {
			if valueReferencesFrame(arg, tainted) || exprPublishesFrame(arg, declared, tainted) {
				return true
			}
		}
		return false
	case *ir.SQLiteCallExpr:
		for _, arg := range expr.Args {
			if valueReferencesFrame(arg, tainted) || exprPublishesFrame(arg, declared, tainted) {
				return true
			}
		}
		return false
	case *ir.BinaryExpr:
		return exprPublishesFrame(expr.Left, declared, tainted) || exprPublishesFrame(expr.Right, declared, tainted)
	case *ir.UnaryExpr:
		return exprPublishesFrame(expr.Operand, declared, tainted)
	case *ir.LenExpr:
		return exprPublishesFrame(expr.Value, declared, tainted)
	case *ir.IndexExpr:
		return exprPublishesFrame(expr.Collection, declared, tainted) || exprPublishesFrame(expr.Index, declared, tainted)
	case *ir.SliceExpr:
		return exprPublishesFrame(expr.Collection, declared, tainted) || exprPublishesFrame(expr.Start, declared, tainted) || exprPublishesFrame(expr.End, declared, tainted)
	case *ir.CloneExpr:
		return exprPublishesFrame(expr.Value, declared, tainted)
	case *ir.MakeExpr:
		return exprPublishesFrame(expr.Len, declared, tainted)
	case *ir.ArrayLiteral:
		for _, elem := range expr.Elements {
			if exprPublishesFrame(elem, declared, tainted) {
				return true
			}
		}
		return false
	case *ir.ListLiteral:
		for _, elem := range expr.Elements {
			if exprPublishesFrame(elem, declared, tainted) {
				return true
			}
		}
		return false
	case *ir.ReadFileExpr, *ir.ImageDimensionExpr, *ir.CreateDirExpr, *ir.RemoveExpr, *ir.ReadDirExpr, *ir.PathQueryExpr:
		return exprPublishesFrame(exprPath(expr), declared, tainted)
	case *ir.ReadCSVExpr:
		return exprPublishesFrame(expr.Path, declared, tainted) || exprPublishesFrame(expr.Columns, declared, tainted)
	case *ir.ReadPPMExpr:
		return exprPublishesFrame(expr.Path, declared, tainted)
	case *ir.HTTPRequestStringExpr:
		return exprPublishesFrame(expr.Request, declared, tainted) || exprPublishesFrame(expr.Name, declared, tainted)
	case *ir.GPUAllocExpr:
		return exprPublishesFrame(expr.Len, declared, tainted)
	case *ir.GPUFromExpr:
		return exprPublishesFrame(expr.Host, declared, tainted)
	default:
		return true
	}
}

func exprPath(expr ir.Expr) ir.Expr {
	switch expr := expr.(type) {
	case *ir.ReadFileExpr:
		return expr.Path
	case *ir.CreateDirExpr:
		return expr.Path
	case *ir.RemoveExpr:
		return expr.Path
	case *ir.ReadDirExpr:
		return expr.Path
	case *ir.PathQueryExpr:
		return expr.Path
	case *ir.ImageDimensionExpr:
		return expr.Path
	default:
		return nil
	}
}

func valueReferencesFrame(expr ir.Expr, tainted map[string]bool) bool {
	if expr == nil {
		return false
	}
	switch expr := expr.(type) {
	case *ir.IdentExpr:
		return tainted[expr.Name]
	case *ir.IntLiteral, *ir.FloatLiteral, *ir.StringLiteral, *ir.BoolLiteral, *ir.NilLiteral, *ir.GPUCoordExpr, *ir.TimeNowExpr, *ir.ReadIntExpr, *ir.ReadFloatExpr, *ir.ReadBoolExpr, *ir.ImageDimensionExpr, *ir.GPUAllocExpr, *ir.GPUFromExpr:
		return false
	case *ir.BinaryExpr:
		if expr.Operator == "+" && ast.TypeEqual(expr.Type(), ast.StringType) {
			return true
		}
		return false
	case *ir.UnaryExpr:
		return expr.Operator == "&" || isDynamicType(expr.Type())
	case *ir.LenExpr, *ir.PathQueryExpr:
		return false
	case *ir.ArrayLiteral, *ir.ListLiteral, *ir.StructLiteral, *ir.MapLiteral, *ir.NewExpr, *ir.KeysExpr, *ir.ValuesExpr, *ir.MakeExpr, *ir.CloneExpr, *ir.ReadLineExpr, *ir.ReadFileExpr, *ir.ReadCSVExpr, *ir.ReadPPMExpr, *ir.HTTPRequestStringExpr, *ir.CreateDirExpr, *ir.RemoveExpr, *ir.ReadDirExpr:
		return true
	case *ir.FieldExpr:
		if !isDynamicType(expr.Type()) && !ast.TypeEqual(expr.Type(), ast.StringType) {
			return false
		}
		return valueReferencesFrame(expr.Object, tainted)
	case *ir.MapLookupExpr:
		return valueReferencesFrame(expr.Map, tainted)
	case *ir.CallExpr, *ir.SQLiteCallExpr:
		return isDynamicType(expr.Type())
	case *ir.IndexExpr, *ir.SliceExpr:
		return dynamicValueReferencesCollection(expr, tainted)
	default:
		return true
	}
}

func dynamicValueReferencesCollection(expr ir.Expr, tainted map[string]bool) bool {
	switch expr := expr.(type) {
	case *ir.IndexExpr:
		if !isDynamicType(expr.Type()) && !ast.TypeEqual(expr.Type(), ast.StringType) {
			return false
		}
		return valueReferencesFrame(expr.Collection, tainted)
	case *ir.SliceExpr:
		return valueReferencesFrame(expr.Collection, tainted)
	default:
		return true
	}
}

func copyStringSet(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for key, value := range in {
		if value {
			out[key] = true
		}
	}
	return out
}

type emittedExpr struct {
	prelude []string
	value   string
}

type collectionFamily struct {
	name    string
	cType   string
	cloneFn string
}

type Options struct {
	SourceRoot string
}

func Generate(program *ir.Program) (string, error) {
	return GenerateWithOptions(program, Options{})
}

func GenerateWithOptions(program *ir.Program, opts Options) (string, error) {
	var out bytes.Buffer

	if program.UsesHTTP {
		out.WriteString("#ifndef _GNU_SOURCE\n#define _GNU_SOURCE\n#endif\n")
	}
	if len(program.Kernels) > 0 {
		out.WriteString("#define TRUX_RUNTIME_CUDA 1\n")
	}
	fmt.Fprintf(&out, "#include %s\n", cStringLiteral(runtimec.HeaderName))
	if program.UsesHTTP {
		fmt.Fprintf(&out, "#include %s\n", cStringLiteral(runtimec.HTTPHeaderName))
	}
	if program.UsesSQLite {
		fmt.Fprintf(&out, "#include %s\n", cStringLiteral(runtimec.SQLiteHeaderName))
	}
	fmt.Fprintln(&out)

	if err := emitProgramTypes(&out, program); err != nil {
		return "", err
	}

	tuples, err := collectTupleTypes(program)
	if err != nil {
		return "", err
	}
	if len(tuples) > 0 {
		for _, tuple := range tuples {
			tName, err := tupleTypeName(tuple)
			if err != nil {
				return "", err
			}
			if runtimeTuples[tName] {
				continue
			}
			fmt.Fprintf(&out, "typedef struct {\n")
			for i, elem := range tuple.Types {
				cTyp, err := emitType(elem)
				if err != nil {
					return "", err
				}
				fmt.Fprintf(&out, "    %s f%d;\n", cTyp, i)
			}
			fmt.Fprintf(&out, "} %s;\n\n", tName)
		}
	}

	if program.UsesSQLite {
		fmt.Fprintf(&out, "\n%s\n", runtimec.SQLiteImplSource)
	}

	for _, fn := range program.Functions {
		if err := emitPrototype(&out, fn); err != nil {
			return "", err
		}
	}
	if len(program.Kernels) > 0 {
		for _, kernel := range program.Kernels {
			if err := emitKernelPrototype(&out, kernel); err != nil {
				return "", err
			}
		}
	}
	fmt.Fprintln(&out)

	for _, kernel := range program.Kernels {
		if err := emitKernelFunc(&out, kernel, opts); err != nil {
			return "", err
		}
		fmt.Fprintln(&out)
	}

	for _, fn := range program.Functions {
		if err := emitFunc(&out, program, fn, opts); err != nil {
			return "", err
		}
		fmt.Fprintln(&out)
	}

	fmt.Fprintln(&out, "int main(void) {")
	fmt.Fprintln(&out, "    rt_stdio_binary();")
	fmt.Fprintln(&out, "    rt_arena trux_arena;")
	fmt.Fprintln(&out, "    rt_arena_init(&trux_arena);")
	fmt.Fprintln(&out, "    rt_context trux_ctx = {&trux_arena, NULL};")
	fmt.Fprintf(&out, "    int64_t trux_exit_code = %s(&trux_ctx, &trux_arena);\n", mangleFunc("main"))
	if program.UsesSQLite {
		fmt.Fprintln(&out, "    rt_sqlite_cleanup_all();")
	}
	fmt.Fprintln(&out, "    rt_context_free_chunks(&trux_ctx);")
	fmt.Fprintln(&out, "    rt_arena_deinit(&trux_arena);")
	fmt.Fprintln(&out, "    return (int)trux_exit_code;")
	fmt.Fprintln(&out, "}")

	return out.String(), nil
}

func emitPrototype(out *bytes.Buffer, fn *ir.Func) error {
	return emitSignature(out, fn, true)
}

func emitFunc(out *bytes.Buffer, program *ir.Program, fn *ir.Func, opts Options) error {
	emitLine(out, "", fn.Pos, opts.SourceRoot)
	if err := emitSignature(out, fn, false); err != nil {
		return err
	}
	returnType, err := emitType(fn.ReturnType)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, " {")
	fmt.Fprintln(out, "    (void)trux_ctx;")
	fmt.Fprintln(out, "    (void)trux_result_arena;")

	needsFrame := functionNeedsFrame(fn)
	scratch := resultArena
	if needsFrame {
		fmt.Fprintln(out, "    rt_arena trux_frame;")
		fmt.Fprintln(out, "    rt_arena_init_frame(&trux_frame, trux_ctx);")
		scratch = frameArena
	}
	fmt.Fprintf(out, "    %s trux_return_value;\n", returnType)
	emitter := &funcEmitter{program: program, opts: opts, fn: fn, scratch: scratch}
	for _, stmt := range fn.Body {
		if err := emitter.emitStmt(out, stmt, 1, opts); err != nil {
			return err
		}
	}
	fmt.Fprintln(out, "    rt_runtime_fail(\"missing return\");")
	fmt.Fprintln(out, "trux_return:")
	if needsFrame {
		fmt.Fprintln(out, "    rt_arena_deinit(&trux_frame);")
	}
	fmt.Fprintln(out, "    return trux_return_value;")
	fmt.Fprintln(out, "}")
	return nil
}

func emitKernelSignature(out *bytes.Buffer, fn *ir.Func, prototype bool) error {
	fmt.Fprintf(out, "__global__ void %s(", mangleKernel(fn.Name))
	sep := false
	writeParam := func(typ string, name string) {
		if sep {
			fmt.Fprint(out, ", ")
		}
		sep = true
		fmt.Fprintf(out, "%s %s", typ, name)
	}
	if fn.Injected >= 1 {
		writeParam("int64_t", "trux_dim_x")
	}
	if fn.Injected == 2 {
		writeParam("int64_t", "trux_dim_y")
	}
	for i, param := range fn.Params {
		if i < fn.Injected {
			continue
		}
		paramType, err := emitKernelType(param.Type)
		if err != nil {
			return err
		}
		writeParam(paramType, mangleIdent(param.Name))
	}
	fmt.Fprint(out, ")")
	if prototype {
		fmt.Fprintln(out, ";")
	}
	return nil
}

func emitKernelPrototype(out *bytes.Buffer, fn *ir.Func) error {
	return emitKernelSignature(out, fn, true)
}

func emitKernelFunc(out *bytes.Buffer, fn *ir.Func, opts Options) error {
	emitLine(out, "", fn.Pos, opts.SourceRoot)
	if err := emitKernelSignature(out, fn, false); err != nil {
		return err
	}
	fmt.Fprintln(out, " {")
	bodyLevel := 1
	if fn.Injected == 1 {
		name := mangleIdent(fn.Params[0].Name)
		fmt.Fprintf(out, "    int64_t %s = %s;\n", name, "((int64_t)(blockIdx.x * blockDim.x + threadIdx.x))")
		fmt.Fprintf(out, "    if (%s < trux_dim_x) {\n", name)
		bodyLevel = 2
	} else if fn.Injected == 2 {
		row := mangleIdent(fn.Params[0].Name)
		col := mangleIdent(fn.Params[1].Name)
		fmt.Fprintf(out, "    int64_t %s = %s;\n", row, "((int64_t)(blockIdx.y * blockDim.y + threadIdx.y))")
		fmt.Fprintf(out, "    int64_t %s = %s;\n", col, "((int64_t)(blockIdx.x * blockDim.x + threadIdx.x))")
		fmt.Fprintf(out, "    if (%s < trux_dim_y && %s < trux_dim_x) {\n", row, col)
		bodyLevel = 2
	}
	if err := emitKernelStmts(out, fn.Body, bodyLevel, opts, &loopStack{}); err != nil {
		return err
	}
	if fn.Injected > 0 {
		fmt.Fprintln(out, "    }")
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

func (e *funcEmitter) emitStmt(out *bytes.Buffer, stmt ir.Stmt, level int, opts Options) error {
	indent := strings.Repeat("    ", level)
	emitLine(out, indent, stmtPos(stmt), opts.SourceRoot)
	switch stmt := stmt.(type) {
	case *ir.LetStmt:
		if len(stmt.Bindings) > 1 {
			val, err := e.emitOrderedExpr(stmt.Value, e.scratch)
			if err != nil {
				return err
			}
			e.writePrelude(out, indent, val.prelude)
			hasNonBlank := false
			for _, target := range stmt.Bindings {
				if target.Name != "_" {
					hasNonBlank = true
					break
				}
			}
			if !hasNonBlank {
				fmt.Fprintf(out, "%s(void)(%s);\n", indent, val.value)
				return nil
			}
			tupleType, err := emitType(stmt.Value.Type())
			if err != nil {
				return err
			}
			tmpName := fmt.Sprintf("trux_tmp_%d", e.temp)
			e.temp++
			fmt.Fprintf(out, "%s%s %s = %s;\n", indent, tupleType, tmpName, val.value)
			for i, binding := range stmt.Bindings {
				if binding.Name == "_" {
					continue
				}
				typ, err := emitType(binding.Type)
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "%s%s %s = %s.f%d;\n", indent, typ, mangleIdent(binding.Name), tmpName, i)
			}
			return nil
		}
		if stmt.Name == "_" {
			val, err := e.emitOrderedExpr(stmt.Value, e.scratch)
			if err != nil {
				return err
			}
			e.writePrelude(out, indent, val.prelude)
			fmt.Fprintf(out, "%s(void)(%s);\n", indent, val.value)
			return nil
		}
		typ, err := emitType(stmt.Type)
		if err != nil {
			return err
		}
		value, err := e.emitOrderedExpr(stmt.Value, e.scratch)
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, value.prelude)
		fmt.Fprintf(out, "%s%s %s = %s;\n", indent, typ, mangleIdent(stmt.Name), value.value)
	case *ir.ReturnStmt:
		if len(stmt.Values) == 0 {
			fmt.Fprintf(out, "%sgoto trux_return;\n", indent)
			return nil
		}
		if len(stmt.Values) == 1 {
			targetArena := e.scratch
			_, directClone := stmt.Values[0].(*ir.CloneExpr)
			if directClone {
				targetArena = resultArena
			} else if (ast.TypeEqual(stmt.Values[0].Type(), ast.StringType) || isDynamicType(stmt.Values[0].Type())) && !needsReturnCopyOut(stmt.Values[0]) {
				targetArena = resultArena
			} else if tuple, ok := stmt.Values[0].Type().(*ast.TupleType); ok && isDynamicType(tuple) {
				targetArena = resultArena
			}
			value, err := e.emitOrderedExpr(stmt.Values[0], targetArena)
			if err != nil {
				return err
			}
			e.writePrelude(out, indent, value.prelude)
			result := value.value
			if !directClone && needsReturnCopyOut(stmt.Values[0]) {
				if _, ok := stmt.Values[0].Type().(*ast.TupleType); !ok {
					result, err = emitClone(stmt.Values[0].Type(), resultArena, result)
					if err != nil {
						return err
					}
				}
			}
			fmt.Fprintf(out, "%strux_return_value = %s;\n", indent, result)
			fmt.Fprintf(out, "%sgoto trux_return;\n", indent)
			return nil
		}
		var fieldInits []string
		for _, valExpr := range stmt.Values {
			targetArena := e.scratch
			_, directClone := valExpr.(*ir.CloneExpr)
			if directClone {
				targetArena = resultArena
			} else if ast.TypeEqual(valExpr.Type(), ast.StringType) && !needsReturnCopyOut(valExpr) {
				targetArena = resultArena
			}
			val, err := e.emitOrderedExpr(valExpr, targetArena)
			if err != nil {
				return err
			}
			e.writePrelude(out, indent, val.prelude)
			res := val.value
			if !directClone && needsReturnCopyOut(valExpr) {
				res, err = emitClone(valExpr.Type(), resultArena, res)
				if err != nil {
					return err
				}
			}
			fieldInits = append(fieldInits, res)
		}
		tupleType, err := emitType(e.fn.ReturnType)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%strux_return_value = RT_LIT(%s, %s);\n", indent, tupleType, strings.Join(fieldInits, ", "))
		fmt.Fprintf(out, "%sgoto trux_return;\n", indent)
	case *ir.AssignStmt:
		if len(stmt.Targets) > 1 {
			val, err := e.emitOrderedExpr(stmt.Value, e.scratch)
			if err != nil {
				return err
			}
			e.writePrelude(out, indent, val.prelude)
			hasNonBlank := false
			for _, target := range stmt.Targets {
				if target.Name != "_" {
					hasNonBlank = true
					break
				}
			}
			if !hasNonBlank {
				fmt.Fprintf(out, "%s(void)(%s);\n", indent, val.value)
				return nil
			}
			tupleType, err := emitType(stmt.Value.Type())
			if err != nil {
				return err
			}
			tmpName := fmt.Sprintf("trux_tmp_%d", e.temp)
			e.temp++
			fmt.Fprintf(out, "%s%s %s = %s;\n", indent, tupleType, tmpName, val.value)
			for i, target := range stmt.Targets {
				if target.Name == "_" {
					continue
				}
				fmt.Fprintf(out, "%s%s = %s.f%d;\n", indent, mangleIdent(target.Name), tmpName, i)
			}
			return nil
		}
		if stmt.Name == "_" {
			val, err := e.emitOrderedExpr(stmt.Value, e.scratch)
			if err != nil {
				return err
			}
			e.writePrelude(out, indent, val.prelude)
			fmt.Fprintf(out, "%s(void)(%s);\n", indent, val.value)
			return nil
		}
		value, err := e.emitOrderedExpr(stmt.Value, e.scratch)
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, value.prelude)
		fmt.Fprintf(out, "%s%s = %s;\n", indent, mangleIdent(stmt.Name), value.value)
	case *ir.IndexAssignStmt:
		collection, err := e.emitOrderedOperand(stmt.Target.Collection, e.scratch)
		if err != nil {
			return err
		}
		valueArena := collectionValueArena(stmt.Target.Collection)
		indexArena := e.scratch
		if _, isMap := stmt.Target.Collection.Type().(*ast.MapType); isMap {
			indexArena = valueArena
		}
		index, err := e.emitOrderedOperand(stmt.Target.Index, indexArena)
		if err != nil {
			return err
		}
		value, err := e.emitOrderedOperand(stmt.Value, valueArena)
		if err != nil {
			return err
		}
		setter, err := collectionHelper("set", stmt.Target.Collection.Type())
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, collection.prelude)
		e.writePrelude(out, indent, index.prelude)
		e.writePrelude(out, indent, value.prelude)
		fmt.Fprintf(out, "%s%s(%s, %s, %s);\n", indent, setter, collection.value, index.value, value.value)
	case *ir.FieldAssignStmt:
		target, err := e.emitLValue(stmt.Target, e.scratch)
		if err != nil {
			return err
		}
		valueArena := lvalueValueArena(stmt.Target)
		value, err := e.emitOrderedOperand(stmt.Value, valueArena)
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, target.prelude)
		e.writePrelude(out, indent, value.prelude)
		fmt.Fprintf(out, "%s%s = %s;\n", indent, target.value, value.value)
	case *ir.DerefAssignStmt:
		target, err := e.emitOrderedOperand(stmt.Target, e.scratch)
		if err != nil {
			return err
		}
		valueArena := collectionValueArena(stmt.Target)
		value, err := e.emitOrderedOperand(stmt.Value, valueArena)
		if err != nil {
			return err
		}
		elemCType, err := emitType(stmt.Value.Type())
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, target.prelude)
		e.writePrelude(out, indent, value.prelude)
		fmt.Fprintf(out, "%s(*((%s*)rt_check_not_null_at(%s, %s, %d, %d))) = %s;\n",
			indent,
			elemCType,
			target.value,
			cStringLiteral(sourcePath(stmt.Pos.File, opts.SourceRoot)),
			stmt.Pos.Line,
			stmt.Pos.Column,
			value.value,
		)
	case *ir.DeleteStmt:
		mapVal, err := e.emitOrderedOperand(stmt.Map, e.scratch)
		if err != nil {
			return err
		}
		keyVal, err := e.emitOrderedOperand(stmt.Key, e.scratch)
		if err != nil {
			return err
		}
		mapName, err := mapRuntimeName(stmt.MapType.(*ast.MapType))
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, mapVal.prelude)
		e.writePrelude(out, indent, keyVal.prelude)
		fmt.Fprintf(out, "%srt_map_%s_delete(%s, %s);\n", indent, mapName, mapVal.value, keyVal.value)
	case *ir.IfStmt:
		condition, err := e.emitCondition(stmt.Condition)
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, condition.prelude)
		fmt.Fprintf(out, "%sif (%s) {\n", indent, condition.value)
		if err := e.emitStmts(out, stmt.Then, level+1, opts); err != nil {
			return err
		}
		if len(stmt.Else) == 0 {
			fmt.Fprintf(out, "%s}\n", indent)
			break
		}
		fmt.Fprintf(out, "%s} else {\n", indent)
		if err := e.emitStmts(out, stmt.Else, level+1, opts); err != nil {
			return err
		}
		fmt.Fprintf(out, "%s}\n", indent)
	case *ir.ForStmt:
		fmt.Fprintf(out, "%s{\n", indent)
		if stmt.Init != nil {
			if err := e.emitStmt(out, stmt.Init, level+1, opts); err != nil {
				return err
			}
		}
		// Only the function frame can be rewound. trux_result_arena belongs to the caller.
		if e.scratch == frameArena && forLoopCanRewind(stmt) {
			mark := fmt.Sprintf("trux_loop_mark_%d", e.marks)
			e.marks++
			fmt.Fprintf(out, "%s    for (;;) {\n", indent)
			fmt.Fprintf(out, "%s        rt_arena_mark %s = rt_arena_mark_current(%s);\n", indent, mark, e.scratch)
			if stmt.Condition != nil {
				condition, err := e.emitCondition(stmt.Condition)
				if err != nil {
					return err
				}
				e.writePrelude(out, indent+"        ", condition.prelude)
				fmt.Fprintf(out, "%s        if (!(%s)) {\n", indent, condition.value)
				fmt.Fprintf(out, "%s            rt_arena_rewind(%s, %s);\n", indent, e.scratch, mark)
				fmt.Fprintf(out, "%s            break;\n", indent)
				fmt.Fprintf(out, "%s        }\n", indent)
			}
			label := e.pushLoop()
			if err := e.emitStmts(out, stmt.Body, level+2, opts); err != nil {
				return err
			}
			if loopHasContinue(stmt.Body) {
				fmt.Fprintf(out, "%s%s:\n", indent+"        ", label)
				if stmt.Post == nil {
					fmt.Fprintf(out, "%s(void)0;\n", indent+"        ")
				}
			}
			if stmt.Post != nil {
				if err := e.emitStmt(out, stmt.Post, level+2, opts); err != nil {
					return err
				}
			}
			fmt.Fprintf(out, "%s        rt_arena_rewind(%s, %s);\n", indent, e.scratch, mark)
			e.popLoop()
			fmt.Fprintf(out, "%s    }\n", indent)
			fmt.Fprintf(out, "%s}\n", indent)
			break
		}
		if stmt.Condition == nil {
			fmt.Fprintf(out, "%s    for (;;) {\n", indent)
		} else {
			condition, err := e.emitCondition(stmt.Condition)
			if err != nil {
				return err
			}
			if len(condition.prelude) == 0 {
				fmt.Fprintf(out, "%s    for (; %s;) {\n", indent, condition.value)
			} else {
				fmt.Fprintf(out, "%s    for (;;) {\n", indent)
				e.writePrelude(out, indent+"        ", condition.prelude)
				fmt.Fprintf(out, "%s        if (!(%s)) {\n", indent, condition.value)
				fmt.Fprintf(out, "%s            break;\n", indent)
				fmt.Fprintf(out, "%s        }\n", indent)
			}
		}
		label := e.pushLoop()
		if err := e.emitStmts(out, stmt.Body, level+2, opts); err != nil {
			return err
		}
		if loopHasContinue(stmt.Body) {
			fmt.Fprintf(out, "%s%s:\n", indent+"        ", label)
			if stmt.Post == nil {
				fmt.Fprintf(out, "%s(void)0;\n", indent+"        ")
			}
		}
		if stmt.Post != nil {
			if err := e.emitStmt(out, stmt.Post, level+2, opts); err != nil {
				return err
			}
		}
		e.popLoop()
		fmt.Fprintf(out, "%s    }\n", indent)
		fmt.Fprintf(out, "%s}\n", indent)
	case *ir.BreakStmt:
		fmt.Fprintf(out, "%sbreak;\n", indent)
	case *ir.ContinueStmt:
		fmt.Fprintf(out, "%sgoto %s;\n", indent, e.loops[len(e.loops)-1])
	case *ir.PrintStmt:
		if len(stmt.Args) != len(stmt.Types) {
			return fmt.Errorf("print arg/type count mismatch")
		}
		for i, argExpr := range stmt.Args {
			arg, err := e.emitOrderedExpr(argExpr, e.scratch)
			if err != nil {
				return err
			}
			e.writePrelude(out, indent, arg.prelude)
			switch {
			case ast.TypeEqual(stmt.Types[i], ast.IntType):
				fmt.Fprintf(out, "%srt_print_int(%s);\n", indent, arg.value)
			case ast.TypeEqual(stmt.Types[i], ast.FloatType):
				fmt.Fprintf(out, "%srt_print_float(%s);\n", indent, arg.value)
			case ast.TypeEqual(stmt.Types[i], ast.StringType):
				fmt.Fprintf(out, "%srt_print_string(%s);\n", indent, arg.value)
			case ast.TypeEqual(stmt.Types[i], ast.BoolType):
				fmt.Fprintf(out, "%srt_print_bool(%s);\n", indent, arg.value)
			case ast.TypeEqual(stmt.Types[i], ast.ErrorType):
				fmt.Fprintf(out, "%srt_print_error(%s);\n", indent, arg.value)
			default:
				return fmt.Errorf("unsupported print type %s", stmt.Types[i])
			}
		}
		fmt.Fprintf(out, "%srt_print_newline();\n", indent)
	case *ir.AssertStmt:
		condition, err := e.emitCondition(stmt.Condition)
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, condition.prelude)
		fmt.Fprintf(out, "%sif (!(%s)) {\n", indent, condition.value)
		message, err := e.emitOrderedExpr(stmt.Message, e.scratch)
		if err != nil {
			return err
		}
		e.writePrelude(out, indent+"    ", message.prelude)
		fmt.Fprintf(out, "%s    rt_assert_fail_at(%s, %s, %d, %d);\n",
			indent,
			message.value,
			cStringLiteral(sourcePath(stmt.Pos.File, opts.SourceRoot)),
			stmt.Pos.Line,
			stmt.Pos.Column,
		)
		fmt.Fprintf(out, "%s}\n", indent)
	case *ir.AppendStmt:
		list, err := e.emitOrderedOperand(stmt.List, e.scratch)
		if err != nil {
			return err
		}
		valueArena := collectionValueArena(stmt.List)
		value, err := e.emitOrderedOperand(stmt.Value, valueArena)
		if err != nil {
			return err
		}
		appendFn, err := listHelper("append", stmt.ListType)
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, list.prelude)
		e.writePrelude(out, indent, value.prelude)
		fmt.Fprintf(out, "%s%s(%s, %s);\n", indent, appendFn, list.value, value.value)
	case *ir.WriteFileStmt:
		path, err := e.emitOrderedOperand(stmt.Path, e.scratch)
		if err != nil {
			return err
		}
		contents, err := e.emitOrderedOperand(stmt.Contents, e.scratch)
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, path.prelude)
		e.writePrelude(out, indent, contents.prelude)
		fmt.Fprintf(out, "%srt_write_file(%s, %s);\n", indent, path.value, contents.value)
	case *ir.WriteCSVStmt:
		path, err := e.emitOrderedOperand(stmt.Path, e.scratch)
		if err != nil {
			return err
		}
		cells, err := e.emitOrderedOperand(stmt.Cells, e.scratch)
		if err != nil {
			return err
		}
		columns, err := e.emitOrderedOperand(stmt.Columns, e.scratch)
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, path.prelude)
		e.writePrelude(out, indent, cells.prelude)
		e.writePrelude(out, indent, columns.prelude)
		fmt.Fprintf(out, "%srt_write_csv(%s, %s, %s);\n", indent, path.value, cells.value, columns.value)
	case *ir.WritePPMStmt:
		path, err := e.emitOrderedOperand(stmt.Path, e.scratch)
		if err != nil {
			return err
		}
		pixels, err := e.emitOrderedOperand(stmt.Pixels, e.scratch)
		if err != nil {
			return err
		}
		width, err := e.emitOrderedOperand(stmt.Width, e.scratch)
		if err != nil {
			return err
		}
		height, err := e.emitOrderedOperand(stmt.Height, e.scratch)
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, path.prelude)
		e.writePrelude(out, indent, pixels.prelude)
		e.writePrelude(out, indent, width.prelude)
		e.writePrelude(out, indent, height.prelude)
		fmt.Fprintf(out, "%srt_write_ppm(%s, %s, %s, %s);\n", indent, path.value, pixels.value, width.value, height.value)
	case *ir.GPUCopyStmt:
		first, err := e.emitOrderedOperand(stmt.First, e.scratch)
		if err != nil {
			return err
		}
		second, err := e.emitOrderedOperand(stmt.Second, e.scratch)
		if err != nil {
			return err
		}
		var elem ast.Type
		if stmt.Kind == semtypes.GPUCallCopyToDevice {
			elem = stmt.Second.Type().(*ast.GPUBufferType).Elem
		} else {
			elem = stmt.First.Type().(*ast.GPUBufferType).Elem
		}
		name, err := gpuElemName(elem)
		if err != nil {
			return err
		}
		if stmt.Kind == semtypes.GPUCallCopyToDevice {
			e.writePrelude(out, indent, first.prelude)
			e.writePrelude(out, indent, second.prelude)
			fmt.Fprintf(out, "%srt_gpu_copy_to_device_%s(%s, %s);\n", indent, name, first.value, second.value)
		} else {
			e.writePrelude(out, indent, first.prelude)
			e.writePrelude(out, indent, second.prelude)
			fmt.Fprintf(out, "%srt_gpu_copy_to_host_%s(%s, %s);\n", indent, name, first.value, second.value)
		}
	case *ir.GPUFreeStmt:
		buffer, err := e.emitOrderedOperand(stmt.Buffer, e.scratch)
		if err != nil {
			return err
		}
		name, err := gpuElemName(stmt.Buffer.Type().(*ast.GPUBufferType).Elem)
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, buffer.prelude)
		fmt.Fprintf(out, "%srt_gpu_free_%s(%s);\n", indent, name, buffer.value)
	case *ir.GPUSyncStmt:
		fmt.Fprintf(out, "%srt_gpu_sync();\n", indent)
	case *ir.GPULaunchStmt:
		dims := make([]emittedExpr, 0, len(stmt.Dims))
		for _, dim := range stmt.Dims {
			value, err := e.emitOrderedOperand(dim, e.scratch)
			if err != nil {
				return err
			}
			dims = append(dims, value)
		}
		args := make([]emittedExpr, 0, len(stmt.Args))
		for _, arg := range stmt.Args {
			value, err := e.emitKernelLaunchArg(arg)
			if err != nil {
				return err
			}
			args = append(args, value)
		}
		dimValues := make([]string, 0, len(dims))
		for _, dim := range dims {
			e.writePrelude(out, indent, dim.prelude)
			dimValues = append(dimValues, dim.value)
		}
		argValues := make([]string, 0, len(args))
		for _, arg := range args {
			e.writePrelude(out, indent, arg.prelude)
			argValues = append(argValues, arg.value)
		}
		var launch string
		switch stmt.Axes {
		case 1:
			launch = fmt.Sprintf("%s<<<dim3((unsigned int)(((%s) + 255) / 256), 1u, 1u), dim3(256u, 1u, 1u)>>>(%s)", mangleKernel(stmt.KernelName), dimValues[0], strings.Join(append([]string{dimValues[0]}, argValues...), ", "))
		case 2:
			launch = fmt.Sprintf("%s<<<dim3((unsigned int)(((%s) + 15) / 16), (unsigned int)(((%s) + 15) / 16), 1u), dim3(16u, 16u, 1u)>>>(%s)", mangleKernel(stmt.KernelName), dimValues[0], dimValues[1], strings.Join(append([]string{dimValues[0], dimValues[1]}, argValues...), ", "))
		default:
			launch = fmt.Sprintf("%s<<<dim3((unsigned int)%s, (unsigned int)%s, (unsigned int)%s), dim3((unsigned int)%s, (unsigned int)%s, (unsigned int)%s)>>>(%s)", mangleKernel(stmt.KernelName), dimValues[0], dimValues[1], dimValues[2], dimValues[3], dimValues[4], dimValues[5], strings.Join(argValues, ", "))
		}
		fmt.Fprintf(out, "%s%s;\n", indent, launch)
		fmt.Fprintf(out, "%srt_cuda_check(cudaGetLastError(), \"launch\");\n", indent)
	case *ir.TimeSleepStmt:
		millis, err := e.emitOrderedOperand(stmt.Millis, e.scratch)
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, millis.prelude)
		fmt.Fprintf(out, "%srt_time_sleep_millis(%s);\n", indent, millis.value)
	case *ir.HTTPListenStmt:
		port, err := e.emitOrderedOperand(stmt.Port, e.scratch)
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, port.prelude)
		fmt.Fprintf(out, "%srt_http_listen(%s);\n", indent, port.value)
	case *ir.HTTPServeStmt:
		host, err := e.emitOrderedOperand(stmt.Host, e.scratch)
		if err != nil {
			return err
		}
		port, err := e.emitOrderedOperand(stmt.Port, e.scratch)
		if err != nil {
			return err
		}
		workers, err := e.emitOrderedOperand(stmt.Workers, e.scratch)
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, host.prelude)
		e.writePrelude(out, indent, port.prelude)
		e.writePrelude(out, indent, workers.prelude)
		fmt.Fprintf(out, "%srt_http_serve(%s, %s, %s);\n", indent, host.value, port.value, workers.value)
	case *ir.HTTPUseStmt:
		if stmt.Prefix != nil {
			prefix, err := e.emitOrderedOperand(stmt.Prefix, e.scratch)
			if err != nil {
				return err
			}
			e.writePrelude(out, indent, prefix.prelude)
			fmt.Fprintf(out, "%srt_http_register_middleware(%s, %s);\n", indent, prefix.value, mangleFunc(stmt.HandlerName))
		} else {
			fmt.Fprintf(out, "%srt_http_register_middleware(RT_LIT(rt_string, NULL, 0), %s);\n", indent, mangleFunc(stmt.HandlerName))
		}
	case *ir.HTTPRouteStmt:
		method, err := e.emitOrderedOperand(stmt.Method, e.scratch)
		if err != nil {
			return err
		}
		path, err := e.emitOrderedOperand(stmt.Path, e.scratch)
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, method.prelude)
		e.writePrelude(out, indent, path.prelude)
		fmt.Fprintf(out, "%srt_http_register_route(%s, %s, %s);\n", indent, method.value, path.value, mangleFunc(stmt.HandlerName))
	case *ir.HTTPNotFoundStmt:
		fmt.Fprintf(out, "%srt_http_register_not_found(%s);\n", indent, mangleFunc(stmt.HandlerName))
	case *ir.HTTPSetStmt:
		request, err := e.emitOrderedOperand(stmt.Request, e.scratch)
		if err != nil {
			return err
		}
		key, err := e.emitOrderedOperand(stmt.Key, e.scratch)
		if err != nil {
			return err
		}
		val, err := e.emitOrderedOperand(stmt.Val, e.scratch)
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, request.prelude)
		e.writePrelude(out, indent, key.prelude)
		e.writePrelude(out, indent, val.prelude)
		fmt.Fprintf(out, "%srt_http_set_context(%s, %s, %s);\n", indent, request.value, key.value, val.value)
	case *ir.HTTPSetHeaderStmt:
		request, err := e.emitOrderedOperand(stmt.Request, e.scratch)
		if err != nil {
			return err
		}
		name, err := e.emitOrderedOperand(stmt.Name, e.scratch)
		if err != nil {
			return err
		}
		val, err := e.emitOrderedOperand(stmt.Val, e.scratch)
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, request.prelude)
		e.writePrelude(out, indent, name.prelude)
		e.writePrelude(out, indent, val.prelude)
		fmt.Fprintf(out, "%srt_http_set_header(%s, %s, %s);\n", indent, request.value, name.value, val.value)
	case *ir.HTTPTextStmt:
		request, err := e.emitOrderedOperand(stmt.Request, e.scratch)
		if err != nil {
			return err
		}
		status, err := e.emitOrderedOperand(stmt.Status, e.scratch)
		if err != nil {
			return err
		}
		body, err := e.emitOrderedOperand(stmt.Body, e.scratch)
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, request.prelude)
		e.writePrelude(out, indent, status.prelude)
		e.writePrelude(out, indent, body.prelude)
		fmt.Fprintf(out, "%srt_http_text(%s, %s, %s);\n", indent, request.value, status.value, body.value)
	case *ir.HTTPJSONStmt:
		request, err := e.emitOrderedOperand(stmt.Request, e.scratch)
		if err != nil {
			return err
		}
		status, err := e.emitOrderedOperand(stmt.Status, e.scratch)
		if err != nil {
			return err
		}
		body, err := e.emitOrderedOperand(stmt.Body, e.scratch)
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, request.prelude)
		e.writePrelude(out, indent, status.prelude)
		e.writePrelude(out, indent, body.prelude)
		fmt.Fprintf(out, "%srt_http_json(%s, %s, %s);\n", indent, request.value, status.value, body.value)
	case *ir.HTTPStatusStmt:
		request, err := e.emitOrderedOperand(stmt.Request, e.scratch)
		if err != nil {
			return err
		}
		status, err := e.emitOrderedOperand(stmt.Status, e.scratch)
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, request.prelude)
		e.writePrelude(out, indent, status.prelude)
		fmt.Fprintf(out, "%srt_http_status(%s, %s);\n", indent, request.value, status.value)
	case *ir.HTTPRespondStmt:
		request, err := e.emitOrderedOperand(stmt.Request, e.scratch)
		if err != nil {
			return err
		}
		status, err := e.emitOrderedOperand(stmt.Status, e.scratch)
		if err != nil {
			return err
		}
		contentType, err := e.emitOrderedOperand(stmt.ContentType, e.scratch)
		if err != nil {
			return err
		}
		body, err := e.emitOrderedOperand(stmt.Body, e.scratch)
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, request.prelude)
		e.writePrelude(out, indent, status.prelude)
		e.writePrelude(out, indent, contentType.prelude)
		e.writePrelude(out, indent, body.prelude)
		fmt.Fprintf(out, "%srt_http_respond(%s, %s, %s, %s);\n", indent, request.value, status.value, contentType.value, body.value)
	case *ir.ExprStmt:
		expr, err := e.emitOrderedExpr(stmt.Expr, e.scratch)
		if err != nil {
			return err
		}
		e.writePrelude(out, indent, expr.prelude)
		fmt.Fprintf(out, "%s%s;\n", indent, expr.value)
	default:
		return fmt.Errorf("unsupported IR statement %T", stmt)
	}

	return nil
}

func (e *funcEmitter) emitStmts(out *bytes.Buffer, stmts []ir.Stmt, level int, opts Options) error {
	for _, stmt := range stmts {
		if err := e.emitStmt(out, stmt, level, opts); err != nil {
			return err
		}
	}

	return nil
}

func (e *funcEmitter) writePrelude(out *bytes.Buffer, indent string, lines []string) {
	for _, line := range lines {
		fmt.Fprintf(out, "%s%s\n", indent, line)
	}
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

func lvalueValueArena(expr ir.Expr) string {
	for {
		switch e := expr.(type) {
		case *ir.FieldExpr:
			if _, ok := e.Object.Type().(*ast.PointerType); ok {
				return collectionValueArena(e.Object)
			}
			expr = e.Object
		case *ir.IndexExpr:
			return collectionValueArena(e.Collection)
		case *ir.UnaryExpr:
			if e.Operator == "*" {
				return collectionValueArena(e.Operand)
			}
			return collectionValueArena(e)
		default:
			return collectionValueArena(expr)
		}
	}
}

func (e *funcEmitter) emitLValue(expr ir.Expr, targetArena string) (emittedExpr, error) {
	switch expr := expr.(type) {
	case *ir.IdentExpr:
		return emittedExpr{value: mangleIdent(expr.Name)}, nil
	case *ir.UnaryExpr:
		if expr.Operator == "*" {
			op, err := e.emitOrderedOperand(expr.Operand, targetArena)
			if err != nil {
				return emittedExpr{}, err
			}
			elemCType, err := emitType(expr.Type())
			if err != nil {
				return emittedExpr{}, err
			}
			val := fmt.Sprintf("(*((%s*)rt_check_not_null_at(%s, %s, %d, %d)))",
				elemCType,
				op.value,
				cStringLiteral(sourcePath(expr.Pos.File, e.opts.SourceRoot)),
				expr.Pos.Line,
				expr.Pos.Column,
			)
			return emittedExpr{prelude: op.prelude, value: val}, nil
		}
	case *ir.FieldExpr:
		if ptrType, ok := expr.Object.Type().(*ast.PointerType); ok {
			obj, err := e.emitOrderedOperand(expr.Object, targetArena)
			if err != nil {
				return emittedExpr{}, err
			}
			structCType, err := emitType(ptrType.Elem)
			if err != nil {
				return emittedExpr{}, err
			}
			val := fmt.Sprintf("(((%s*)rt_check_not_null_at(%s, %s, %d, %d))->%s)",
				structCType,
				obj.value,
				cStringLiteral(sourcePath(expr.Pos.File, e.opts.SourceRoot)),
				expr.Pos.Line,
				expr.Pos.Column,
				mangleIdent(expr.Field),
			)
			return emittedExpr{prelude: obj.prelude, value: val}, nil
		}
		obj, err := e.emitLValue(expr.Object, targetArena)
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: obj.prelude, value: fmt.Sprintf("(%s.%s)", obj.value, mangleIdent(expr.Field))}, nil
	case *ir.IndexExpr:
		collection, err := e.emitOrderedOperand(expr.Collection, targetArena)
		if err != nil {
			return emittedExpr{}, err
		}
		index, err := e.emitOrderedOperand(expr.Index, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		addrFn, err := collectionHelper("addr", expr.Collection.Type())
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: joinPreludes(collection, index), value: fmt.Sprintf("(*%s(%s, %s))", addrFn, collection.value, index.value)}, nil
	}
	return e.emitOrderedOperand(expr, targetArena)
}

func (e *funcEmitter) emitDefaultValue(typ ast.Type, targetArena string) (string, error) {
	return emitDefaultValueForProgram(e.program, typ, targetArena)
}

func emitDefaultValueForProgram(program *ir.Program, typ ast.Type, targetArena string) (string, error) {
	switch t := typ.(type) {
	case ast.ScalarType:
		switch {
		case ast.TypeEqual(t, ast.IntType):
			return "0", nil
		case ast.TypeEqual(t, ast.FloatType):
			return "0.0", nil
		case ast.TypeEqual(t, ast.BoolType):
			return "false", nil
		case ast.TypeEqual(t, ast.StringType):
			return "RT_LIT(rt_string, NULL, 0)", nil
		case ast.TypeEqual(t, ast.ErrorType):
			return "NULL", nil
		default:
			return "", fmt.Errorf("unsupported default value type %s", typ)
		}
	case *ast.PointerType:
		return "NULL", nil
	case *ast.SliceType:
		cType, err := emitType(t)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("RT_LIT(%s, NULL, 0)", cType), nil
	case *ast.ArrayType:
		cType, err := emitType(t)
		if err != nil {
			return "", err
		}
		elemCType, err := emitType(t.Elem)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("RT_LIT(%s, (%s*)rt_arena_alloc_count(%s, %d, sizeof(%s), true), %d)", cType, elemCType, targetArena, t.Length, elemCType, t.Length), nil
	case *ast.ListType:
		newFn, err := listHelper("new", t)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s(%s, 0)", newFn, targetArena), nil
	case *ast.MapType:
		mapName, err := mapRuntimeName(t)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("rt_map_%s_new(%s, 0)", mapName, targetArena), nil
	case *ast.StructType:
		cType, err := emitType(t)
		if err != nil {
			return "", err
		}
		if program == nil {
			return fmt.Sprintf("RT_LIT(%s, 0)", cType), nil
		}
		sig, ok := program.Structs[t.TargetName()]
		if !ok || len(sig.Fields) == 0 {
			return fmt.Sprintf("RT_LIT(%s, 0)", cType), nil
		}
		var vals []string
		for _, f := range sig.Fields {
			v, err := emitDefaultValueForProgram(program, f.Type, targetArena)
			if err != nil {
				return "", err
			}
			vals = append(vals, v)
		}
		return fmt.Sprintf("RT_LIT(%s, %s)", cType, strings.Join(vals, ", ")), nil
	default:
		return "", fmt.Errorf("unsupported default value type %s", typ)
	}
}

func emitCondition(expr ir.Expr) (string, error) {
	condition, err := emitExpr(expr, frameArena)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(condition, "(") && strings.HasSuffix(condition, ")") {
		return condition[1 : len(condition)-1], nil
	}
	return condition, nil
}

func (e *funcEmitter) emitCondition(expr ir.Expr) (emittedExpr, error) {
	condition, err := e.emitOrderedExpr(expr, e.scratch)
	if err != nil {
		return emittedExpr{}, err
	}
	condition.value = trimParens(condition.value)
	return condition, nil
}

func (e *funcEmitter) emitOrderedOperand(expr ir.Expr, targetArena string) (emittedExpr, error) {
	value, err := e.emitOrderedExpr(expr, targetArena)
	if err != nil {
		return emittedExpr{}, err
	}
	if !exprHasSideEffect(expr) {
		return value, nil
	}

	tmp, err := e.newTemp(expr.Type(), value.value)
	if err != nil {
		return emittedExpr{}, err
	}
	value.prelude = append(value.prelude, tmp.decl)
	value.value = tmp.name
	return value, nil
}

func (e *funcEmitter) emitOrderedExpr(expr ir.Expr, targetArena string) (emittedExpr, error) {
	switch expr := expr.(type) {
	case *ir.ArrayLiteral:
		values, err := e.emitOrderedValueArray(expr.Elements, targetArena)
		if err != nil {
			return emittedExpr{}, err
		}
		ctor, err := collectionHelper("from_values", expr.Type())
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: values.prelude, value: fmt.Sprintf("%s(%s, %s, %d)", ctor, targetArena, values.value, len(expr.Elements))}, nil
	case *ir.ListLiteral:
		values, err := e.emitOrderedValueArray(expr.Elements, targetArena)
		if err != nil {
			return emittedExpr{}, err
		}
		ctor, err := listHelper("from_values", expr.Type())
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: values.prelude, value: fmt.Sprintf("%s(%s, %s, %d)", ctor, targetArena, values.value, len(expr.Elements))}, nil
	case *ir.StructLiteral:
		st := expr.Type().(*ast.StructType)
		cType, err := emitType(st)
		if err != nil {
			return emittedExpr{}, err
		}
		sig, ok := e.program.Structs[st.TargetName()]
		if !ok {
			return emittedExpr{}, fmt.Errorf("unknown struct %s", st.TargetName())
		}
		provided := make(map[string]string, len(expr.Fields))
		var prelude []string
		for _, f := range expr.Fields {
			val, err := e.emitOrderedOperand(f.Value, targetArena)
			if err != nil {
				return emittedExpr{}, err
			}
			prelude = append(prelude, val.prelude...)
			provided[f.Name] = val.value
		}
		if len(sig.Fields) == 0 {
			return emittedExpr{prelude: prelude, value: fmt.Sprintf("RT_LIT(%s, 0)", cType)}, nil
		}
		fieldVals := make([]string, 0, len(sig.Fields))
		for _, declField := range sig.Fields {
			if val, ok := provided[declField.Name]; ok {
				fieldVals = append(fieldVals, val)
			} else {
				defVal, err := e.emitDefaultValue(declField.Type, targetArena)
				if err != nil {
					return emittedExpr{}, err
				}
				fieldVals = append(fieldVals, defVal)
			}
		}
		return emittedExpr{prelude: prelude, value: fmt.Sprintf("RT_LIT(%s, %s)", cType, strings.Join(fieldVals, ", "))}, nil
	case *ir.MapLiteral:
		mt := expr.Type().(*ast.MapType)
		mapName, err := mapRuntimeName(mt)
		if err != nil {
			return emittedExpr{}, err
		}
		tmp, err := e.newTemp(expr.Type(), fmt.Sprintf("rt_map_%s_new(%s, %d)", mapName, targetArena, len(expr.Entries)))
		if err != nil {
			return emittedExpr{}, err
		}
		prelude := []string{tmp.decl}
		for _, entry := range expr.Entries {
			k, err := e.emitOrderedOperand(entry.Key, targetArena)
			if err != nil {
				return emittedExpr{}, err
			}
			v, err := e.emitOrderedOperand(entry.Value, targetArena)
			if err != nil {
				return emittedExpr{}, err
			}
			prelude = append(prelude, k.prelude...)
			prelude = append(prelude, v.prelude...)
			prelude = append(prelude, fmt.Sprintf("rt_map_%s_set(%s, %s, %s);", mapName, tmp.name, k.value, v.value))
		}
		return emittedExpr{prelude: prelude, value: tmp.name}, nil
	case *ir.NewExpr:
		elemCType, err := emitType(expr.ElemType)
		if err != nil {
			return emittedExpr{}, err
		}
		defVal, err := e.emitDefaultValue(expr.ElemType, targetArena)
		if err != nil {
			return emittedExpr{}, err
		}
		ptrName := fmt.Sprintf("trux_tmp_%d", e.temp)
		e.temp++
		prelude := []string{
			fmt.Sprintf("%s* %s = (%s*)rt_arena_alloc(%s, sizeof(%s));", elemCType, ptrName, elemCType, targetArena, elemCType),
			fmt.Sprintf("*%s = %s;", ptrName, defVal),
		}
		return emittedExpr{prelude: prelude, value: ptrName}, nil
	case *ir.MakeExpr:
		if mt, ok := expr.Type().(*ast.MapType); ok {
			mapName, err := mapRuntimeName(mt)
			if err != nil {
				return emittedExpr{}, err
			}
			if expr.Len == nil {
				return emittedExpr{value: fmt.Sprintf("rt_map_%s_new(%s, 0)", mapName, targetArena)}, nil
			}
			length, err := e.emitOrderedOperand(expr.Len, e.scratch)
			if err != nil {
				return emittedExpr{}, err
			}
			return emittedExpr{prelude: length.prelude, value: fmt.Sprintf("rt_map_%s_new(%s, %s)", mapName, targetArena, length.value)}, nil
		}
		length, err := e.emitOrderedOperand(expr.Len, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		makeFn, err := sliceMakeHelper(expr.Type())
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: length.prelude, value: fmt.Sprintf("%s(%s, %s)", makeFn, targetArena, length.value)}, nil
	case *ir.CloneExpr:
		value, err := e.emitOrderedOperand(expr.Value, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		cloned, err := emitClone(expr.Value.Type(), targetArena, value.value)
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: value.prelude, value: cloned}, nil
	case *ir.ReadFileExpr:
		path, err := e.emitOrderedOperand(expr.Path, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: path.prelude, value: fmt.Sprintf("rt_read_file(%s, %s)", targetArena, path.value)}, nil
	case *ir.CreateDirExpr:
		path, err := e.emitOrderedOperand(expr.Path, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: path.prelude, value: fmt.Sprintf("rt_create_dir(%s, %s)", targetArena, path.value)}, nil
	case *ir.RemoveExpr:
		path, err := e.emitOrderedOperand(expr.Path, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: path.prelude, value: fmt.Sprintf("rt_remove(%s, %s)", targetArena, path.value)}, nil
	case *ir.ReadDirExpr:
		path, err := e.emitOrderedOperand(expr.Path, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: path.prelude, value: fmt.Sprintf("rt_read_dir(%s, %s)", targetArena, path.value)}, nil
	case *ir.PathQueryExpr:
		path, err := e.emitOrderedOperand(expr.Path, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		switch expr.Kind {
		case semtypes.IOCallExists:
			return emittedExpr{prelude: path.prelude, value: fmt.Sprintf("rt_exists(%s)", path.value)}, nil
		case semtypes.IOCallIsFile:
			return emittedExpr{prelude: path.prelude, value: fmt.Sprintf("rt_is_file(%s)", path.value)}, nil
		case semtypes.IOCallIsDir:
			return emittedExpr{prelude: path.prelude, value: fmt.Sprintf("rt_is_dir(%s)", path.value)}, nil
		default:
			return emittedExpr{}, fmt.Errorf("unsupported path query expression %s", expr.Kind)
		}
	case *ir.ErrorsNewExpr:
		msg, err := e.emitOrderedOperand(expr.Message, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: msg.prelude, value: fmt.Sprintf("rt_error_new(%s, %s)", targetArena, msg.value)}, nil
	case *ir.ErrorsMessageExpr:
		errVal, err := e.emitOrderedOperand(expr.Err, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: errVal.prelude, value: fmt.Sprintf("rt_error_message(%s)", errVal.value)}, nil
	case *ir.ReadCSVExpr:
		path, err := e.emitOrderedOperand(expr.Path, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		columns, err := e.emitOrderedOperand(expr.Columns, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: joinPreludes(path, columns), value: fmt.Sprintf("rt_read_csv(%s, %s, %s)", targetArena, path.value, columns.value)}, nil
	case *ir.WriteFileExpr:
		path, err := e.emitOrderedOperand(expr.Path, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		contents, err := e.emitOrderedOperand(expr.Contents, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: joinPreludes(path, contents), value: fmt.Sprintf("rt_write_file(%s, %s, %s)", targetArena, path.value, contents.value)}, nil
	case *ir.WriteCSVExpr:
		path, err := e.emitOrderedOperand(expr.Path, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		cells, err := e.emitOrderedOperand(expr.Cells, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		columns, err := e.emitOrderedOperand(expr.Columns, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: joinPreludes(path, cells, columns), value: fmt.Sprintf("rt_write_csv(%s, %s, %s, %s)", targetArena, path.value, cells.value, columns.value)}, nil
	case *ir.WritePPMExpr:
		path, err := e.emitOrderedOperand(expr.Path, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		pixels, err := e.emitOrderedOperand(expr.Pixels, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		width, err := e.emitOrderedOperand(expr.Width, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		height, err := e.emitOrderedOperand(expr.Height, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: joinPreludes(path, pixels, width, height), value: fmt.Sprintf("rt_write_ppm(%s, %s, %s, %s, %s)", targetArena, path.value, pixels.value, width.value, height.value)}, nil
	case *ir.ImageDimensionExpr:
		path, err := e.emitOrderedOperand(expr.Path, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		switch expr.Kind {
		case semtypes.IOCallImageWidth:
			return emittedExpr{prelude: path.prelude, value: fmt.Sprintf("rt_image_width(%s)", path.value)}, nil
		case semtypes.IOCallImageHeight:
			return emittedExpr{prelude: path.prelude, value: fmt.Sprintf("rt_image_height(%s)", path.value)}, nil
		default:
			return emittedExpr{}, fmt.Errorf("unsupported image dimension expression %s", expr.Kind)
		}
	case *ir.ReadPPMExpr:
		path, err := e.emitOrderedOperand(expr.Path, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: path.prelude, value: fmt.Sprintf("rt_read_ppm(%s, %s)", targetArena, path.value)}, nil
	case *ir.HTTPRequestStringExpr:
		request, err := e.emitOrderedOperand(expr.Request, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		switch expr.Kind {
		case semtypes.HTTPCallMethod:
			return emittedExpr{prelude: request.prelude, value: fmt.Sprintf("rt_http_method(%s, %s)", targetArena, request.value)}, nil
		case semtypes.HTTPCallPath:
			return emittedExpr{prelude: request.prelude, value: fmt.Sprintf("rt_http_path(%s, %s)", targetArena, request.value)}, nil
		case semtypes.HTTPCallQuery:
			return emittedExpr{prelude: request.prelude, value: fmt.Sprintf("rt_http_query(%s, %s)", targetArena, request.value)}, nil
		case semtypes.HTTPCallBody:
			return emittedExpr{prelude: request.prelude, value: fmt.Sprintf("rt_http_body(%s, %s)", targetArena, request.value)}, nil
		case semtypes.HTTPCallHeader:
			name, err := e.emitOrderedOperand(expr.Name, e.scratch)
			if err != nil {
				return emittedExpr{}, err
			}
			return emittedExpr{prelude: joinPreludes(request, name), value: fmt.Sprintf("rt_http_header(%s, %s, %s)", targetArena, request.value, name.value)}, nil
		case semtypes.HTTPCallParam:
			name, err := e.emitOrderedOperand(expr.Name, e.scratch)
			if err != nil {
				return emittedExpr{}, err
			}
			return emittedExpr{prelude: joinPreludes(request, name), value: fmt.Sprintf("rt_http_param(%s, %s, %s)", targetArena, request.value, name.value)}, nil
		case semtypes.HTTPCallContext:
			name, err := e.emitOrderedOperand(expr.Name, e.scratch)
			if err != nil {
				return emittedExpr{}, err
			}
			return emittedExpr{prelude: joinPreludes(request, name), value: fmt.Sprintf("rt_http_context(%s, %s, %s)", targetArena, request.value, name.value)}, nil
		default:
			return emittedExpr{}, fmt.Errorf("unsupported HTTP request string expression %s", expr.Kind)
		}
	case *ir.GPUAllocExpr:
		length, err := e.emitOrderedOperand(expr.Len, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		buf := expr.Type().(*ast.GPUBufferType)
		name, err := gpuElemName(buf.Elem)
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: length.prelude, value: fmt.Sprintf("rt_gpu_alloc_%s(%s)", name, length.value)}, nil
	case *ir.GPUFromExpr:
		host, err := e.emitOrderedOperand(expr.Host, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		buf := expr.Type().(*ast.GPUBufferType)
		elem, err := gpuElemName(buf.Elem)
		if err != nil {
			return emittedExpr{}, err
		}
		tmp, err := e.newTemp(expr.Type(), fmt.Sprintf("rt_gpu_alloc_%s((int64_t)%s.len)", elem, host.value))
		if err != nil {
			return emittedExpr{}, err
		}
		prelude := append(append([]string{}, host.prelude...), tmp.decl)
		prelude = append(prelude, fmt.Sprintf("rt_gpu_copy_to_device_%s(%s, %s);", elem, host.value, tmp.name))
		return emittedExpr{prelude: prelude, value: tmp.name}, nil
	case *ir.CallExpr:
		args := make([]emittedExpr, 0, len(expr.Args)+1)
		callResultArena := e.scratch
		if isDynamicType(expr.Type()) {
			callResultArena = targetArena
		}
		for _, arg := range expr.Args {
			argArena := e.scratch
			if isDynamicType(expr.Type()) && isDynamicType(arg.Type()) {
				argArena = callResultArena
			}
			emitArg, err := e.emitOrderedOperand(arg, argArena)
			if err != nil {
				return emittedExpr{}, err
			}
			args = append(args, emitArg)
		}
		values := []string{"trux_ctx", callResultArena}
		var prelude []string
		for _, arg := range args {
			prelude = append(prelude, arg.prelude...)
			values = append(values, arg.value)
		}
		return emittedExpr{prelude: prelude, value: fmt.Sprintf("%s(%s)", mangleFunc(expr.Callee), strings.Join(values, ", "))}, nil
	case *ir.SQLiteCallExpr:
		args := make([]emittedExpr, 0, len(expr.Args))
		callResultArena := e.scratch
		if isDynamicType(expr.Type()) {
			callResultArena = targetArena
		}
		for _, arg := range expr.Args {
			argArena := e.scratch
			if isDynamicType(expr.Type()) && isDynamicType(arg.Type()) {
				argArena = callResultArena
			}
			emitArg, err := e.emitOrderedOperand(arg, argArena)
			if err != nil {
				return emittedExpr{}, err
			}
			args = append(args, emitArg)
		}
		values := []string{callResultArena}
		var prelude []string
		for _, arg := range args {
			prelude = append(prelude, arg.prelude...)
			values = append(values, arg.value)
		}
		fnName := sqliteRuntimeFnName(expr.Kind)
		return emittedExpr{prelude: prelude, value: fmt.Sprintf("%s(%s)", fnName, strings.Join(values, ", "))}, nil
	case *ir.UnaryExpr:
		if expr.Operator == "&" {
			if _, ok := expr.Operand.(*ir.StructLiteral); ok {
				lit, err := e.emitOrderedExpr(expr.Operand, targetArena)
				if err != nil {
					return emittedExpr{}, err
				}
				elemCType, err := emitType(expr.Operand.Type())
				if err != nil {
					return emittedExpr{}, err
				}
				ptrName := fmt.Sprintf("trux_tmp_%d", e.temp)
				e.temp++
				prelude := append([]string{}, lit.prelude...)
				prelude = append(prelude,
					fmt.Sprintf("%s* %s = (%s*)rt_arena_alloc(%s, sizeof(%s));", elemCType, ptrName, elemCType, targetArena, elemCType),
					fmt.Sprintf("*%s = %s;", ptrName, lit.value),
				)
				return emittedExpr{prelude: prelude, value: ptrName}, nil
			}
			lval, err := e.emitLValue(expr.Operand, targetArena)
			if err != nil {
				return emittedExpr{}, err
			}
			return emittedExpr{prelude: lval.prelude, value: fmt.Sprintf("(&(%s))", lval.value)}, nil
		}
		if expr.Operator == "*" {
			operand, err := e.emitOrderedOperand(expr.Operand, targetArena)
			if err != nil {
				return emittedExpr{}, err
			}
			elemCType, err := emitType(expr.Type())
			if err != nil {
				return emittedExpr{}, err
			}
			val := fmt.Sprintf("(*((%s*)rt_check_not_null_at(%s, %s, %d, %d)))",
				elemCType,
				operand.value,
				cStringLiteral(sourcePath(expr.Pos.File, e.opts.SourceRoot)),
				expr.Pos.Line,
				expr.Pos.Column,
			)
			return emittedExpr{prelude: operand.prelude, value: val}, nil
		}
		operand, err := e.emitOrderedOperand(expr.Operand, targetArena)
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: operand.prelude, value: emitUnaryValue(expr.Operator, expr.Type(), operand.value)}, nil
	case *ir.FieldExpr:
		obj, err := e.emitOrderedOperand(expr.Object, targetArena)
		if err != nil {
			return emittedExpr{}, err
		}
		if ptrType, ok := expr.Object.Type().(*ast.PointerType); ok {
			structCType, err := emitType(ptrType.Elem)
			if err != nil {
				return emittedExpr{}, err
			}
			val := fmt.Sprintf("(((%s*)rt_check_not_null_at(%s, %s, %d, %d))->%s)",
				structCType,
				obj.value,
				cStringLiteral(sourcePath(expr.Pos.File, e.opts.SourceRoot)),
				expr.Pos.Line,
				expr.Pos.Column,
				mangleIdent(expr.Field),
			)
			return emittedExpr{prelude: obj.prelude, value: val}, nil
		}
		return emittedExpr{prelude: obj.prelude, value: fmt.Sprintf("(%s.%s)", obj.value, mangleIdent(expr.Field))}, nil
	case *ir.MapLookupExpr:
		mVal, err := e.emitOrderedOperand(expr.Map, targetArena)
		if err != nil {
			return emittedExpr{}, err
		}
		kVal, err := e.emitOrderedOperand(expr.Key, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		mapName, err := mapRuntimeName(expr.MapType)
		if err != nil {
			return emittedExpr{}, err
		}
		valCType, err := emitType(expr.MapType.Value)
		if err != nil {
			return emittedExpr{}, err
		}
		tupleCType, err := emitType(expr.Type())
		if err != nil {
			return emittedExpr{}, err
		}
		okName := fmt.Sprintf("trux_ok_%d", e.temp)
		valName := fmt.Sprintf("trux_val_%d", e.temp)
		tmpName := fmt.Sprintf("trux_tmp_%d", e.temp)
		e.temp++
		prelude := joinPreludes(mVal, kVal)
		prelude = append(prelude,
			fmt.Sprintf("bool %s = false;", okName),
			fmt.Sprintf("%s %s = rt_map_%s_get_ok(%s, %s, &%s);", valCType, valName, mapName, mVal.value, kVal.value, okName),
			fmt.Sprintf("%s %s = RT_LIT(%s, %s, %s);", tupleCType, tmpName, tupleCType, valName, okName),
		)
		return emittedExpr{prelude: prelude, value: tmpName}, nil
	case *ir.KeysExpr:
		mVal, err := e.emitOrderedOperand(expr.Map, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		mapName, err := mapRuntimeName(expr.MapType)
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: mVal.prelude, value: fmt.Sprintf("rt_map_%s_keys(%s, %s)", mapName, targetArena, mVal.value)}, nil
	case *ir.ValuesExpr:
		mVal, err := e.emitOrderedOperand(expr.Map, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		mapName, err := mapRuntimeName(expr.MapType)
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: mVal.prelude, value: fmt.Sprintf("rt_map_%s_values(%s, %s)", mapName, targetArena, mVal.value)}, nil
	case *ir.BinaryExpr:
		if expr.Operator == "&&" || expr.Operator == "||" {
			return e.emitShortCircuit(expr, targetArena)
		}
		left, err := e.emitOrderedOperand(expr.Left, targetArena)
		if err != nil {
			return emittedExpr{}, err
		}
		right, err := e.emitOrderedOperand(expr.Right, targetArena)
		if err != nil {
			return emittedExpr{}, err
		}
		value, err := emitBinaryExprValue(expr, targetArena, left.value, right.value)
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: joinPreludes(left, right), value: value}, nil
	case *ir.LenExpr:
		value, err := e.emitOrderedOperand(expr.Value, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		if ast.TypeEqual(expr.Value.Type(), ast.StringType) {
			return emittedExpr{prelude: value.prelude, value: fmt.Sprintf("rt_checked_len_i64(%s.len)", value.value)}, nil
		}
		if _, ok := expr.Value.Type().(*ast.ListType); ok {
			return emittedExpr{prelude: value.prelude, value: fmt.Sprintf("rt_checked_len_i64(%s->len)", value.value)}, nil
		}
		if mt, ok := expr.Value.Type().(*ast.MapType); ok {
			mapName, err := mapRuntimeName(mt)
			if err != nil {
				return emittedExpr{}, err
			}
			return emittedExpr{prelude: value.prelude, value: fmt.Sprintf("rt_map_%s_len(%s)", mapName, value.value)}, nil
		}
		return emittedExpr{prelude: value.prelude, value: fmt.Sprintf("rt_checked_len_i64(%s.len)", value.value)}, nil
	case *ir.IndexExpr:
		collection, err := e.emitOrderedOperand(expr.Collection, targetArena)
		if err != nil {
			return emittedExpr{}, err
		}
		index, err := e.emitOrderedOperand(expr.Index, e.scratch)
		if err != nil {
			return emittedExpr{}, err
		}
		if ast.TypeEqual(expr.Collection.Type(), ast.StringType) {
			return emittedExpr{prelude: joinPreludes(collection, index), value: fmt.Sprintf("rt_string_index(%s, %s)", collection.value, index.value)}, nil
		}
		getter, err := collectionHelper("get", expr.Collection.Type())
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: joinPreludes(collection, index), value: fmt.Sprintf("%s(%s, %s)", getter, collection.value, index.value)}, nil
	case *ir.SliceExpr:
		collection, err := e.emitOrderedOperand(expr.Collection, targetArena)
		if err != nil {
			return emittedExpr{}, err
		}
		hasStart, start, err := e.emitOrderedOptionalBound(expr.Start)
		if err != nil {
			return emittedExpr{}, err
		}
		hasEnd, end, err := e.emitOrderedOptionalBound(expr.End)
		if err != nil {
			return emittedExpr{}, err
		}
		prelude := append([]string{}, collection.prelude...)
		prelude = append(prelude, start.prelude...)
		prelude = append(prelude, end.prelude...)
		if ast.TypeEqual(expr.Collection.Type(), ast.StringType) {
			return emittedExpr{prelude: prelude, value: fmt.Sprintf("rt_string_slice(%s, %s, %s, %s, %s)", collection.value, hasStart, start.value, hasEnd, end.value)}, nil
		}
		sliceFn, err := collectionHelper("slice", expr.Collection.Type())
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{prelude: prelude, value: fmt.Sprintf("%s(%s, %s, %s, %s, %s)", sliceFn, collection.value, hasStart, start.value, hasEnd, end.value)}, nil
	default:
		value, err := emitExpr(expr, targetArena)
		if err != nil {
			return emittedExpr{}, err
		}
		return emittedExpr{value: value}, nil
	}
}

type tempExpr struct {
	name string
	decl string
}

func (e *funcEmitter) emitShortCircuit(expr *ir.BinaryExpr, targetArena string) (emittedExpr, error) {
	left, err := e.emitOrderedOperand(expr.Left, targetArena)
	if err != nil {
		return emittedExpr{}, err
	}
	right, err := e.emitOrderedOperand(expr.Right, targetArena)
	if err != nil {
		return emittedExpr{}, err
	}
	tmp, err := e.newTemp(ast.BoolType, left.value)
	if err != nil {
		return emittedExpr{}, err
	}
	cond := tmp.name
	if expr.Operator == "||" {
		cond = "!(" + tmp.name + ")"
	}
	prelude := append([]string{}, left.prelude...)
	prelude = append(prelude, tmp.decl)
	prelude = append(prelude, "if ("+cond+") {")
	for _, line := range right.prelude {
		prelude = append(prelude, "    "+line)
	}
	prelude = append(prelude, "    "+tmp.name+" = "+right.value+";")
	prelude = append(prelude, "}")
	return emittedExpr{prelude: prelude, value: tmp.name}, nil
}

func (e *funcEmitter) newTemp(typ ast.Type, value string) (tempExpr, error) {
	cType, err := emitType(typ)
	if err != nil {
		return tempExpr{}, err
	}
	name := fmt.Sprintf("trux_tmp_%d", e.temp)
	e.temp++
	return tempExpr{
		name: name,
		decl: fmt.Sprintf("%s %s = %s;", cType, name, value),
	}, nil
}

func joinPreludes(values ...emittedExpr) []string {
	var prelude []string
	for _, value := range values {
		prelude = append(prelude, value.prelude...)
	}
	return prelude
}

func (e *funcEmitter) emitOrderedValueArray(elements []ir.Expr, targetArena string) (emittedExpr, error) {
	if len(elements) == 0 {
		return emittedExpr{value: "NULL"}, nil
	}
	elemType := elements[0].Type()
	cType, err := emitType(elemType)
	if err != nil {
		return emittedExpr{}, err
	}
	values := make([]string, 0, len(elements))
	var prelude []string
	for _, elem := range elements {
		value, err := e.emitOrderedOperand(elem, targetArena)
		if err != nil {
			return emittedExpr{}, err
		}
		prelude = append(prelude, value.prelude...)
		values = append(values, value.value)
	}
	return emittedExpr{prelude: prelude, value: fmt.Sprintf("RT_ARRAY(%s, %d, %s)", cType, len(values), strings.Join(values, ", "))}, nil
}

func (e *funcEmitter) emitOrderedOptionalBound(expr ir.Expr) (string, emittedExpr, error) {
	if expr == nil {
		return "false", emittedExpr{value: "0"}, nil
	}
	value, err := e.emitOrderedOperand(expr, e.scratch)
	if err != nil {
		return "", emittedExpr{}, err
	}
	return "true", value, nil
}

func (e *funcEmitter) emitKernelLaunchArg(expr ir.Expr) (emittedExpr, error) {
	value, err := e.emitOrderedOperand(expr, e.scratch)
	if err != nil {
		return emittedExpr{}, err
	}
	if _, ok := expr.Type().(*ast.GPUBufferType); ok {
		value.value += ".data"
	}
	return value, nil
}

func exprHasSideEffect(expr ir.Expr) bool {
	switch expr := expr.(type) {
	case *ir.ReadLineExpr, *ir.ReadIntExpr, *ir.ReadFloatExpr, *ir.ReadBoolExpr,
		*ir.ReadFileExpr, *ir.ReadCSVExpr, *ir.ImageDimensionExpr, *ir.ReadPPMExpr,
		*ir.TimeNowExpr, *ir.HTTPRequestStringExpr, *ir.GPUAllocExpr, *ir.GPUFromExpr, *ir.CallExpr, *ir.SQLiteCallExpr,
		*ir.ErrorsNewExpr, *ir.WriteFileExpr, *ir.WriteCSVExpr, *ir.WritePPMExpr,
		*ir.CreateDirExpr, *ir.RemoveExpr, *ir.ReadDirExpr, *ir.PathQueryExpr,
		*ir.StructLiteral, *ir.MapLiteral, *ir.NewExpr, *ir.MapLookupExpr, *ir.KeysExpr, *ir.ValuesExpr:
		return true
	case *ir.FieldExpr:
		if _, isPtr := expr.Object.Type().(*ast.PointerType); isPtr {
			return true
		}
		return exprHasSideEffect(expr.Object)
	case *ir.ArrayLiteral:
		for _, elem := range expr.Elements {
			if exprHasSideEffect(elem) {
				return true
			}
		}
	case *ir.ListLiteral:
		for _, elem := range expr.Elements {
			if exprHasSideEffect(elem) {
				return true
			}
		}
	case *ir.MakeExpr:
		return exprHasSideEffect(expr.Len)
	case *ir.CloneExpr:
		return exprHasSideEffect(expr.Value)
	case *ir.UnaryExpr:
		if expr.Operator == "*" || expr.Operator == "&" {
			return true
		}
		return exprHasSideEffect(expr.Operand)
	case *ir.BinaryExpr:
		return exprHasSideEffect(expr.Left) || exprHasSideEffect(expr.Right)
	case *ir.LenExpr:
		return exprHasSideEffect(expr.Value)
	case *ir.IndexExpr:
		return exprHasSideEffect(expr.Collection) || exprHasSideEffect(expr.Index)
	case *ir.SliceExpr:
		return exprHasSideEffect(expr.Collection) || exprHasSideEffect(expr.Start) || exprHasSideEffect(expr.End)
	}
	return false
}

func emitExpr(expr ir.Expr, targetArena string) (string, error) {
	switch expr := expr.(type) {
	case *ir.IdentExpr:
		return mangleIdent(expr.Name), nil
	case *ir.IntLiteral:
		return expr.Value, nil
	case *ir.FloatLiteral:
		return expr.Value, nil
	case *ir.StringLiteral:
		return fmt.Sprintf("RT_LIT(rt_string, (const uint8_t*)%s, %d)", cStringLiteral(expr.Value), len(expr.Value)), nil
	case *ir.BoolLiteral:
		if expr.Value {
			return "true", nil
		}
		return "false", nil
	case *ir.NilLiteral:
		return "NULL", nil
	case *ir.ArrayLiteral:
		arrayType, ok := expr.Type().(*ast.ArrayType)
		if !ok {
			return "", fmt.Errorf("array literal has non-array type %s", expr.Type())
		}
		values, err := emitValueArray(arrayType.Elem, expr.Elements, targetArena)
		if err != nil {
			return "", err
		}
		ctor, err := collectionHelper("from_values", expr.Type())
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s(%s, %s, %d)", ctor, targetArena, values, len(expr.Elements)), nil
	case *ir.ListLiteral:
		listType, ok := expr.Type().(*ast.ListType)
		if !ok {
			return "", fmt.Errorf("list literal has non-list type %s", expr.Type())
		}
		values, err := emitValueArray(listType.Elem, expr.Elements, targetArena)
		if err != nil {
			return "", err
		}
		ctor, err := listHelper("from_values", expr.Type())
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s(%s, %s, %d)", ctor, targetArena, values, len(expr.Elements)), nil
	case *ir.MakeExpr:
		length, err := emitExpr(expr.Len, frameArena)
		if err != nil {
			return "", err
		}
		makeFn, err := sliceMakeHelper(expr.Type())
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s(%s, %s)", makeFn, targetArena, length), nil
	case *ir.CloneExpr:
		value, err := emitExpr(expr.Value, frameArena)
		if err != nil {
			return "", err
		}
		return emitClone(expr.Value.Type(), targetArena, value)
	case *ir.ReadLineExpr:
		return fmt.Sprintf("rt_read_line(%s)", targetArena), nil
	case *ir.ReadIntExpr:
		return fmt.Sprintf("rt_read_int(%s)", targetArena), nil
	case *ir.ReadFloatExpr:
		return fmt.Sprintf("rt_read_float(%s)", targetArena), nil
	case *ir.ReadBoolExpr:
		return fmt.Sprintf("rt_read_bool(%s)", targetArena), nil
	case *ir.ReadFileExpr:
		path, err := emitExpr(expr.Path, frameArena)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("rt_read_file(%s, %s)", targetArena, path), nil
	case *ir.CreateDirExpr:
		path, err := emitExpr(expr.Path, frameArena)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("rt_create_dir(%s, %s)", targetArena, path), nil
	case *ir.RemoveExpr:
		path, err := emitExpr(expr.Path, frameArena)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("rt_remove(%s, %s)", targetArena, path), nil
	case *ir.ReadDirExpr:
		path, err := emitExpr(expr.Path, frameArena)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("rt_read_dir(%s, %s)", targetArena, path), nil
	case *ir.PathQueryExpr:
		path, err := emitExpr(expr.Path, frameArena)
		if err != nil {
			return "", err
		}
		switch expr.Kind {
		case semtypes.IOCallExists:
			return fmt.Sprintf("rt_exists(%s)", path), nil
		case semtypes.IOCallIsFile:
			return fmt.Sprintf("rt_is_file(%s)", path), nil
		case semtypes.IOCallIsDir:
			return fmt.Sprintf("rt_is_dir(%s)", path), nil
		default:
			return "", fmt.Errorf("unsupported path query expression %s", expr.Kind)
		}
	case *ir.ErrorsNewExpr:
		msg, err := emitExpr(expr.Message, targetArena)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("rt_error_new(%s, %s)", targetArena, msg), nil
	case *ir.ErrorsMessageExpr:
		errVal, err := emitExpr(expr.Err, targetArena)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("rt_error_message(%s)", errVal), nil
	case *ir.ReadCSVExpr:
		path, err := emitExpr(expr.Path, frameArena)
		if err != nil {
			return "", err
		}
		columns, err := emitExpr(expr.Columns, frameArena)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("rt_read_csv(%s, %s, %s)", targetArena, path, columns), nil
	case *ir.WriteFileExpr:
		path, err := emitExpr(expr.Path, frameArena)
		if err != nil {
			return "", err
		}
		contents, err := emitExpr(expr.Contents, frameArena)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("rt_write_file(%s, %s, %s)", targetArena, path, contents), nil
	case *ir.WriteCSVExpr:
		path, err := emitExpr(expr.Path, frameArena)
		if err != nil {
			return "", err
		}
		cells, err := emitExpr(expr.Cells, frameArena)
		if err != nil {
			return "", err
		}
		columns, err := emitExpr(expr.Columns, frameArena)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("rt_write_csv(%s, %s, %s, %s)", targetArena, path, cells, columns), nil
	case *ir.WritePPMExpr:
		path, err := emitExpr(expr.Path, frameArena)
		if err != nil {
			return "", err
		}
		pixels, err := emitExpr(expr.Pixels, frameArena)
		if err != nil {
			return "", err
		}
		width, err := emitExpr(expr.Width, frameArena)
		if err != nil {
			return "", err
		}
		height, err := emitExpr(expr.Height, frameArena)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("rt_write_ppm(%s, %s, %s, %s, %s)", targetArena, path, pixels, width, height), nil
	case *ir.ImageDimensionExpr:
		path, err := emitExpr(expr.Path, frameArena)
		if err != nil {
			return "", err
		}
		switch expr.Kind {
		case semtypes.IOCallImageWidth:
			return fmt.Sprintf("rt_image_width(%s)", path), nil
		case semtypes.IOCallImageHeight:
			return fmt.Sprintf("rt_image_height(%s)", path), nil
		default:
			return "", fmt.Errorf("unsupported image dimension expression %s", expr.Kind)
		}
	case *ir.ReadPPMExpr:
		path, err := emitExpr(expr.Path, frameArena)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("rt_read_ppm(%s, %s)", targetArena, path), nil
	case *ir.TimeNowExpr:
		switch expr.Kind {
		case semtypes.IOCallTimeNowUnixMillis:
			return "rt_time_now_unix_millis()", nil
		case semtypes.IOCallTimeMonotonicNanos:
			return "rt_time_monotonic_nanos()", nil
		default:
			return "", fmt.Errorf("unsupported time expression %s", expr.Kind)
		}
	case *ir.HTTPRequestStringExpr:
		request, err := emitExpr(expr.Request, frameArena)
		if err != nil {
			return "", err
		}
		switch expr.Kind {
		case semtypes.HTTPCallMethod:
			return fmt.Sprintf("rt_http_method(%s, %s)", targetArena, request), nil
		case semtypes.HTTPCallPath:
			return fmt.Sprintf("rt_http_path(%s, %s)", targetArena, request), nil
		case semtypes.HTTPCallQuery:
			return fmt.Sprintf("rt_http_query(%s, %s)", targetArena, request), nil
		case semtypes.HTTPCallBody:
			return fmt.Sprintf("rt_http_body(%s, %s)", targetArena, request), nil
		case semtypes.HTTPCallHeader:
			name, err := emitExpr(expr.Name, frameArena)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("rt_http_header(%s, %s, %s)", targetArena, request, name), nil
		case semtypes.HTTPCallParam:
			name, err := emitExpr(expr.Name, frameArena)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("rt_http_param(%s, %s, %s)", targetArena, request, name), nil
		case semtypes.HTTPCallContext:
			name, err := emitExpr(expr.Name, frameArena)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("rt_http_context(%s, %s, %s)", targetArena, request, name), nil
		default:
			return "", fmt.Errorf("unsupported HTTP request string expression %s", expr.Kind)
		}
	case *ir.GPUAllocExpr:
		length, err := emitExpr(expr.Len, frameArena)
		if err != nil {
			return "", err
		}
		buf := expr.Type().(*ast.GPUBufferType)
		name, err := gpuElemName(buf.Elem)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("rt_gpu_alloc_%s(%s)", name, length), nil
	case *ir.GPUFromExpr:
		return "", fmt.Errorf("gpu.from must be emitted in statement order")
	case *ir.GPUCoordExpr:
		return emitGPUCoord(expr.Kind)
	case *ir.CallExpr:
		args := make([]string, 0, len(expr.Args)+1)
		callResultArena := frameArena
		if isDynamicType(expr.Type()) {
			callResultArena = targetArena
		}
		args = append(args, "trux_ctx", callResultArena)
		for _, arg := range expr.Args {
			argArena := frameArena
			if isDynamicType(expr.Type()) && ast.TypeEqual(arg.Type(), ast.StringType) {
				argArena = callResultArena
			}
			emitArg, err := emitExpr(arg, argArena)
			if err != nil {
				return "", err
			}
			args = append(args, emitArg)
		}
		return fmt.Sprintf("%s(%s)", mangleFunc(expr.Callee), strings.Join(args, ", ")), nil
	case *ir.SQLiteCallExpr:
		callResultArena := frameArena
		if isDynamicType(expr.Type()) {
			callResultArena = targetArena
		}
		args := []string{callResultArena}
		for _, arg := range expr.Args {
			argArena := frameArena
			if isDynamicType(expr.Type()) && ast.TypeEqual(arg.Type(), ast.StringType) {
				argArena = callResultArena
			}
			emitArg, err := emitExpr(arg, argArena)
			if err != nil {
				return "", err
			}
			args = append(args, emitArg)
		}
		fnName := sqliteRuntimeFnName(expr.Kind)
		return fmt.Sprintf("%s(%s)", fnName, strings.Join(args, ", ")), nil
	case *ir.UnaryExpr:
		operand, err := emitExpr(expr.Operand, targetArena)
		if err != nil {
			return "", err
		}
		return emitUnaryValue(expr.Operator, expr.Type(), operand), nil
	case *ir.BinaryExpr:
		left, err := emitExpr(expr.Left, targetArena)
		if err != nil {
			return "", err
		}
		right, err := emitExpr(expr.Right, targetArena)
		if err != nil {
			return "", err
		}
		return emitBinaryExprValue(expr, targetArena, left, right)
	case *ir.LenExpr:
		value, err := emitExpr(expr.Value, frameArena)
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
		collection, err := emitExpr(expr.Collection, targetArena)
		if err != nil {
			return "", err
		}
		index, err := emitExpr(expr.Index, frameArena)
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
		collection, err := emitExpr(expr.Collection, targetArena)
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

func emitUnaryValue(operator string, typ ast.Type, operand string) string {
	switch operator {
	case "!":
		return fmt.Sprintf("!(%s)", operand)
	case "-":
		if ast.TypeEqual(typ, ast.IntType) {
			return fmt.Sprintf("rt_sub_i64(0, %s)", operand)
		}
		return fmt.Sprintf("-(%s)", operand)
	default:
		return fmt.Sprintf("(%s%s)", operator, operand)
	}
}

func emitBinaryExprValue(expr *ir.BinaryExpr, targetArena string, left string, right string) (string, error) {
	if expr.Operator == "in" {
		return fmt.Sprintf("rt_string_contains(%s, %s)", left, right), nil
	}
	if expr.Operator == "+" && ast.TypeEqual(expr.Left.Type(), ast.StringType) {
		return fmt.Sprintf("rt_string_concat(%s, %s, %s)", targetArena, left, right), nil
	}
	if ast.TypeEqual(expr.Left.Type(), ast.StringType) && (expr.Operator == "==" || expr.Operator == "!=") {
		equal := fmt.Sprintf("rt_string_equal(%s, %s)", left, right)
		if expr.Operator == "!=" {
			return fmt.Sprintf("!%s", equal), nil
		}
		return equal, nil
	}
	if ast.TypeEqual(expr.Type(), ast.IntType) {
		if helper, ok := checkedArithmeticHelper(expr.Operator); ok {
			return fmt.Sprintf("%s(%s, %s)", helper, left, right), nil
		}
	}
	return fmt.Sprintf("(%s %s %s)", left, expr.Operator, right), nil
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
		case ast.TypeEqual(typ, ast.ErrorType):
			return "rt_error", nil
		default:
			return "", fmt.Errorf("unsupported type %s", typ)
		}
	case *ast.TupleType:
		return tupleTypeName(typ)
	case *ast.StructType:
		if typ.TargetName() == "trux_sqlite_db" {
			return "trux_sqlite_db", nil
		}
		if typ.TargetName() == "trux_sqlite_stmt" {
			return "trux_sqlite_stmt", nil
		}
		return mangleStruct(typ.TargetName()), nil
	case *ast.PointerType:
		elem, err := emitType(typ.Elem)
		if err != nil {
			return "", err
		}
		return elem + "*", nil
	case *ast.MapType:
		name, err := mapRuntimeName(typ)
		if err != nil {
			return "", err
		}
		return "rt_map_" + name + "*", nil
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
	case *ast.GPUBufferType:
		name, err := gpuElemName(typ.Elem)
		if err != nil {
			return "", err
		}
		return "rt_gpu_buffer_" + name, nil
	default:
		return "", fmt.Errorf("unsupported type %s", typ)
	}
}

func emitKernelType(typ ast.Type) (string, error) {
	switch typ := typ.(type) {
	case ast.ScalarType:
		return emitType(typ)
	case *ast.GPUBufferType:
		elem, err := emitType(typ.Elem)
		if err != nil {
			return "", err
		}
		return elem + "*", nil
	default:
		return "", fmt.Errorf("unsupported kernel type %s", typ)
	}
}

func emitValueArray(elemType ast.Type, elements []ir.Expr, targetArena string) (string, error) {
	if len(elements) == 0 {
		return "NULL", nil
	}
	cType, err := emitType(elemType)
	if err != nil {
		return "", err
	}
	values := make([]string, 0, len(elements))
	for _, elem := range elements {
		value, err := emitExpr(elem, targetArena)
		if err != nil {
			return "", err
		}
		values = append(values, value)
	}
	return fmt.Sprintf("RT_ARRAY(%s, %d, %s)", cType, len(values), strings.Join(values, ", ")), nil
}

func emitClone(typ ast.Type, targetArena string, value string) (string, error) {
	cloneFn, err := cloneHelper(typ)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s(%s, %s)", cloneFn, targetArena, value), nil
}

func emitOptionalBound(expr ir.Expr) (string, string, error) {
	if expr == nil {
		return "false", "0", nil
	}
	value, err := emitExpr(expr, frameArena)
	if err != nil {
		return "", "", err
	}
	return "true", value, nil
}

type loopStack struct {
	labels []string
	next   int
}

func (s *loopStack) push() string {
	label := fmt.Sprintf("trux_continue_%d", s.next)
	s.next++
	s.labels = append(s.labels, label)
	return label
}

func (s *loopStack) pop() {
	s.labels = s.labels[:len(s.labels)-1]
}

func emitKernelStmts(out *bytes.Buffer, stmts []ir.Stmt, level int, opts Options, loops *loopStack) error {
	for _, stmt := range stmts {
		if err := emitKernelStmt(out, stmt, level, opts, loops); err != nil {
			return err
		}
	}
	return nil
}

func emitKernelStmt(out *bytes.Buffer, stmt ir.Stmt, level int, opts Options, loops *loopStack) error {
	indent := strings.Repeat("    ", level)
	emitLine(out, indent, stmtPos(stmt), opts.SourceRoot)
	switch stmt := stmt.(type) {
	case *ir.LetStmt:
		typ, err := emitKernelType(stmt.Type)
		if err != nil {
			return err
		}
		value, err := emitKernelExpr(stmt.Value)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s%s %s = %s;\n", indent, typ, mangleIdent(stmt.Name), value)
	case *ir.AssignStmt:
		value, err := emitKernelExpr(stmt.Value)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s%s = %s;\n", indent, mangleIdent(stmt.Name), value)
	case *ir.IndexAssignStmt:
		target, err := emitKernelIndex(stmt.Target)
		if err != nil {
			return err
		}
		value, err := emitKernelExpr(stmt.Value)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s%s = %s;\n", indent, target, value)
	case *ir.IfStmt:
		condition, err := emitKernelExpr(stmt.Condition)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%sif (%s) {\n", indent, trimParens(condition))
		if err := emitKernelStmts(out, stmt.Then, level+1, opts, loops); err != nil {
			return err
		}
		if len(stmt.Else) == 0 {
			fmt.Fprintf(out, "%s}\n", indent)
			break
		}
		fmt.Fprintf(out, "%s} else {\n", indent)
		if err := emitKernelStmts(out, stmt.Else, level+1, opts, loops); err != nil {
			return err
		}
		fmt.Fprintf(out, "%s}\n", indent)
	case *ir.ForStmt:
		fmt.Fprintf(out, "%s{\n", indent)
		if stmt.Init != nil {
			if err := emitKernelStmt(out, stmt.Init, level+1, opts, loops); err != nil {
				return err
			}
		}
		if stmt.Condition == nil {
			fmt.Fprintf(out, "%s    for (;;) {\n", indent)
		} else {
			condition, err := emitKernelExpr(stmt.Condition)
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "%s    for (; %s;) {\n", indent, trimParens(condition))
		}
		label := loops.push()
		if err := emitKernelStmts(out, stmt.Body, level+2, opts, loops); err != nil {
			return err
		}
		if loopHasContinue(stmt.Body) {
			fmt.Fprintf(out, "%s%s:\n", indent+"        ", label)
			if stmt.Post == nil {
				fmt.Fprintf(out, "%s(void)0;\n", indent+"        ")
			}
		}
		if stmt.Post != nil {
			if err := emitKernelStmt(out, stmt.Post, level+2, opts, loops); err != nil {
				return err
			}
		}
		loops.pop()
		fmt.Fprintf(out, "%s    }\n", indent)
		fmt.Fprintf(out, "%s}\n", indent)
	case *ir.BreakStmt:
		fmt.Fprintf(out, "%sbreak;\n", indent)
	case *ir.ContinueStmt:
		fmt.Fprintf(out, "%sgoto %s;\n", indent, loops.labels[len(loops.labels)-1])
	default:
		return fmt.Errorf("unsupported kernel statement %T", stmt)
	}
	return nil
}

func emitKernelExpr(expr ir.Expr) (string, error) {
	switch expr := expr.(type) {
	case *ir.IdentExpr:
		return mangleIdent(expr.Name), nil
	case *ir.IntLiteral:
		return expr.Value, nil
	case *ir.FloatLiteral:
		return expr.Value, nil
	case *ir.BoolLiteral:
		if expr.Value {
			return "true", nil
		}
		return "false", nil
	case *ir.UnaryExpr:
		operand, err := emitKernelExpr(expr.Operand)
		if err != nil {
			return "", err
		}
		return emitUnaryValue(expr.Operator, expr.Type(), operand), nil
	case *ir.BinaryExpr:
		left, err := emitKernelExpr(expr.Left)
		if err != nil {
			return "", err
		}
		right, err := emitKernelExpr(expr.Right)
		if err != nil {
			return "", err
		}
		if ast.TypeEqual(expr.Type(), ast.IntType) {
			if helper, ok := checkedArithmeticHelper(expr.Operator); ok {
				return fmt.Sprintf("%s(%s, %s)", helper, left, right), nil
			}
		}
		return fmt.Sprintf("(%s %s %s)", left, expr.Operator, right), nil
	case *ir.IndexExpr:
		return emitKernelIndex(expr)
	case *ir.GPUCoordExpr:
		return emitGPUCoord(expr.Kind)
	default:
		return "", fmt.Errorf("unsupported kernel expression %T", expr)
	}
}

func emitKernelIndex(expr *ir.IndexExpr) (string, error) {
	collection, err := emitKernelExpr(expr.Collection)
	if err != nil {
		return "", err
	}
	index, err := emitKernelExpr(expr.Index)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s[%s]", collection, index), nil
}

func emitKernelLaunchArg(expr ir.Expr) (string, error) {
	value, err := emitExpr(expr, frameArena)
	if err != nil {
		return "", err
	}
	if _, ok := expr.Type().(*ast.GPUBufferType); ok {
		return value + ".data", nil
	}
	return value, nil
}

func emitGPUCoord(kind semtypes.GPUCallKind) (string, error) {
	switch kind {
	case semtypes.GPUCallGlobalX:
		return "((int64_t)(blockIdx.x * blockDim.x + threadIdx.x))", nil
	case semtypes.GPUCallGlobalY:
		return "((int64_t)(blockIdx.y * blockDim.y + threadIdx.y))", nil
	case semtypes.GPUCallGlobalZ:
		return "((int64_t)(blockIdx.z * blockDim.z + threadIdx.z))", nil
	case semtypes.GPUCallThreadX:
		return "((int64_t)threadIdx.x)", nil
	case semtypes.GPUCallThreadY:
		return "((int64_t)threadIdx.y)", nil
	case semtypes.GPUCallThreadZ:
		return "((int64_t)threadIdx.z)", nil
	case semtypes.GPUCallBlockX:
		return "((int64_t)blockIdx.x)", nil
	case semtypes.GPUCallBlockY:
		return "((int64_t)blockIdx.y)", nil
	case semtypes.GPUCallBlockZ:
		return "((int64_t)blockIdx.z)", nil
	case semtypes.GPUCallBlockDimX:
		return "((int64_t)blockDim.x)", nil
	case semtypes.GPUCallBlockDimY:
		return "((int64_t)blockDim.y)", nil
	case semtypes.GPUCallBlockDimZ:
		return "((int64_t)blockDim.z)", nil
	default:
		return "", fmt.Errorf("unsupported GPU coordinate %s", kind)
	}
}

func trimParens(value string) string {
	if strings.HasPrefix(value, "(") && strings.HasSuffix(value, ")") {
		return value[1 : len(value)-1]
	}
	return value
}

func isCollectionType(typ ast.Type) bool {
	_, ok := ast.ElementType(typ)
	return ok
}

func isDynamicType(typ ast.Type) bool {
	if tuple, ok := typ.(*ast.TupleType); ok {
		for _, elem := range tuple.Types {
			if isDynamicType(elem) {
				return true
			}
		}
		return false
	}
	switch typ.(type) {
	case *ast.StructType, *ast.PointerType, *ast.MapType:
		return true
	}
	return ast.TypeEqual(typ, ast.StringType) || ast.TypeEqual(typ, ast.ErrorType) || isCollectionType(typ)
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
	case *ast.MapType:
		name, err := mapRuntimeName(typ)
		if err != nil {
			return "", err
		}
		return "rt_map_" + name + "_" + op, nil
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
	if ast.TypeEqual(typ, ast.ErrorType) {
		return "rt_error_clone", nil
	}
	if isCollectionType(typ) {
		return collectionHelper("clone", typ)
	}
	switch t := typ.(type) {
	case *ast.StructType:
		return mangleStruct(t.TargetName()) + "_clone", nil
	case *ast.PointerType:
		name, err := elemRuntimeName(t)
		if err != nil {
			return "", err
		}
		return "rt_" + name + "_clone", nil
	case *ast.MapType:
		name, err := mapRuntimeName(t)
		if err != nil {
			return "", err
		}
		return "rt_map_" + name + "_clone", nil
	}
	return "", fmt.Errorf("clone does not support %s", typ)
}

func mapRuntimeName(typ *ast.MapType) (string, error) {
	k, err := elemRuntimeName(typ.Key)
	if err != nil {
		return "", err
	}
	v, err := elemRuntimeName(typ.Value)
	if err != nil {
		return "", err
	}
	return k + "_" + v, nil
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
		case ast.TypeEqual(typ, ast.ErrorType):
			return "error", nil
		default:
			return "", fmt.Errorf("unsupported element type %s", typ)
		}
	case *ast.StructType:
		name := typ.TargetName()
		return fmt.Sprintf("struct_%d_%s", len(name), name), nil
	case *ast.PointerType:
		name, err := elemRuntimeName(typ.Elem)
		if err != nil {
			return "", err
		}
		return "ptr_" + name, nil
	case *ast.MapType:
		name, err := mapRuntimeName(typ)
		if err != nil {
			return "", err
		}
		return "map_" + name, nil
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
	case *ast.GPUBufferType:
		name, err := gpuElemName(typ.Elem)
		if err != nil {
			return "", err
		}
		return "gpu_buffer_" + name, nil
	default:
		return "", fmt.Errorf("unsupported element type %s", typ)
	}
}

func isBuiltinCollectionElem(typ ast.Type) bool {
	return ast.TypeEqual(typ, ast.IntType) ||
		ast.TypeEqual(typ, ast.FloatType) ||
		ast.TypeEqual(typ, ast.BoolType) ||
		ast.TypeEqual(typ, ast.StringType)
}

type mapFamily struct {
	typ      *ast.MapType
	name     string
	keyName  string
	keyCType string
	valName  string
	valCType string
	eqFn     string
	hashFn   string
	zeroVal  string
}

func mapKeyHelpers(keyType ast.Type) (string, string, error) {
	switch {
	case ast.TypeEqual(keyType, ast.IntType):
		return "rt_eq_int", "rt_hash_int", nil
	case ast.TypeEqual(keyType, ast.BoolType):
		return "rt_eq_bool", "rt_hash_bool", nil
	case ast.TypeEqual(keyType, ast.StringType):
		return "rt_eq_string", "rt_hash_string", nil
	case isPointerType(keyType):
		return "rt_eq_ptr", "rt_hash_ptr", nil
	default:
		return "", "", fmt.Errorf("unsupported map key type %s", keyType)
	}
}

func isPointerType(typ ast.Type) bool {
	_, ok := typ.(*ast.PointerType)
	return ok
}

func emitStaticZeroValue(program *ir.Program, typ ast.Type) (string, error) {
	switch t := typ.(type) {
	case ast.ScalarType:
		switch {
		case ast.TypeEqual(t, ast.IntType):
			return "0", nil
		case ast.TypeEqual(t, ast.FloatType):
			return "0.0", nil
		case ast.TypeEqual(t, ast.BoolType):
			return "false", nil
		case ast.TypeEqual(t, ast.StringType):
			return "RT_LIT(rt_string, NULL, 0)", nil
		case ast.TypeEqual(t, ast.ErrorType):
			return "NULL", nil
		default:
			return "", fmt.Errorf("unsupported zero value type %s", typ)
		}
	case *ast.PointerType, *ast.ListType, *ast.MapType:
		return "NULL", nil
	case *ast.ArrayType, *ast.SliceType:
		cType, err := emitType(t)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("RT_LIT(%s, NULL, 0)", cType), nil
	case *ast.StructType:
		cType, err := emitType(t)
		if err != nil {
			return "", err
		}
		if program == nil {
			return fmt.Sprintf("RT_LIT(%s, 0)", cType), nil
		}
		sig, ok := program.Structs[t.TargetName()]
		if !ok || len(sig.Fields) == 0 {
			return fmt.Sprintf("RT_LIT(%s, 0)", cType), nil
		}
		vals := make([]string, 0, len(sig.Fields))
		for _, f := range sig.Fields {
			v, err := emitStaticZeroValue(program, f.Type)
			if err != nil {
				return "", err
			}
			vals = append(vals, v)
		}
		return fmt.Sprintf("RT_LIT(%s, %s)", cType, strings.Join(vals, ", ")), nil
	default:
		return "", fmt.Errorf("unsupported zero value type %s", typ)
	}
}

func emitCloneValueExpr(typ ast.Type, arenaExpr string, valueExpr string) (string, error) {
	switch t := typ.(type) {
	case ast.ScalarType:
		if ast.TypeEqual(typ, ast.IntType) || ast.TypeEqual(typ, ast.FloatType) || ast.TypeEqual(typ, ast.BoolType) {
			return valueExpr, nil
		}
	case *ast.GPUBufferType:
		return valueExpr, nil
	case *ast.StructType:
		if t.TargetName() == "trux_sqlite_db" || t.TargetName() == "trux_sqlite_stmt" {
			return valueExpr, nil
		}
	}
	fn, err := cloneHelper(typ)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s(%s, %s)", fn, arenaExpr, valueExpr), nil
}

func emitProgramTypes(out *bytes.Buffer, program *ir.Program) error {
	seenStructs := map[string]bool{}
	var structOrder []semtypes.StructSig
	seenPtrs := map[string]bool{}
	var pointers []*ast.PointerType
	seenCols := map[string]bool{}
	var collections []collectionFamily
	colByName := map[string]collectionFamily{}
	seenMaps := map[string]bool{}
	var maps []mapFamily
	mapByName := map[string]mapFamily{}

	var visitType func(typ ast.Type) error
	ensureCollectionElem := func(elem ast.Type) error {
		if err := visitType(elem); err != nil {
			return err
		}
		if isBuiltinCollectionElem(elem) {
			return nil
		}
		name, err := elemRuntimeName(elem)
		if err != nil {
			return err
		}
		if seenCols[name] {
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
		seenCols[name] = true
		fam := collectionFamily{name: name, cType: cType, cloneFn: cloneFn}
		collections = append(collections, fam)
		colByName[name] = fam
		return nil
	}

	visitStruct := func(name string) error {
		if name == "trux_sqlite_db" || name == "trux_sqlite_stmt" {
			return nil
		}
		if seenStructs[name] {
			return nil
		}
		sig, ok := program.Structs[name]
		if !ok {
			return nil
		}
		seenStructs[name] = true
		for _, f := range sig.Fields {
			if err := visitType(f.Type); err != nil {
				return err
			}
		}
		structOrder = append(structOrder, sig)
		return nil
	}

	visitType = func(typ ast.Type) error {
		if typ == nil {
			return nil
		}
		switch t := typ.(type) {
		case *ast.StructType:
			return visitStruct(t.TargetName())
		case *ast.PointerType:
			if err := visitType(t.Elem); err != nil {
				return err
			}
			name, err := elemRuntimeName(t)
			if err != nil {
				return err
			}
			if !seenPtrs[name] {
				seenPtrs[name] = true
				pointers = append(pointers, t)
			}
			return nil
		case *ast.ArrayType:
			return ensureCollectionElem(t.Elem)
		case *ast.SliceType:
			return ensureCollectionElem(t.Elem)
		case *ast.ListType:
			return ensureCollectionElem(t.Elem)
		case *ast.MapType:
			if err := ensureCollectionElem(t.Key); err != nil {
				return err
			}
			if err := ensureCollectionElem(t.Value); err != nil {
				return err
			}
			mName, err := mapRuntimeName(t)
			if err != nil {
				return err
			}
			if seenMaps[mName] {
				return nil
			}
			kName, err := elemRuntimeName(t.Key)
			if err != nil {
				return err
			}
			kCType, err := emitType(t.Key)
			if err != nil {
				return err
			}
			vName, err := elemRuntimeName(t.Value)
			if err != nil {
				return err
			}
			vCType, err := emitType(t.Value)
			if err != nil {
				return err
			}
			eqFn, hashFn, err := mapKeyHelpers(t.Key)
			if err != nil {
				return err
			}
			zeroVal, err := emitStaticZeroValue(program, t.Value)
			if err != nil {
				return err
			}
			seenMaps[mName] = true
			mf := mapFamily{
				typ:      t,
				name:     mName,
				keyName:  kName,
				keyCType: kCType,
				valName:  vName,
				valCType: vCType,
				eqFn:     eqFn,
				hashFn:   hashFn,
				zeroVal:  zeroVal,
			}
			maps = append(maps, mf)
			mapByName[mName] = mf
			return nil
		case *ast.TupleType:
			for _, elem := range t.Types {
				if err := visitType(elem); err != nil {
					return err
				}
			}
			return nil
		default:
			return nil
		}
	}

	var structNames []string
	for name := range program.Structs {
		structNames = append(structNames, name)
	}
	sort.Strings(structNames)
	for _, name := range structNames {
		if err := visitStruct(name); err != nil {
			return err
		}
	}
	for _, typ := range program.UsedTypes {
		if err := visitType(typ); err != nil {
			return err
		}
	}
	for _, fn := range program.Functions {
		if err := visitType(fn.ReturnType); err != nil {
			return err
		}
		for _, p := range fn.Params {
			if err := visitType(p.Type); err != nil {
				return err
			}
		}
	}

	if len(structOrder) == 0 && len(pointers) == 0 && len(collections) == 0 && len(maps) == 0 {
		return nil
	}

	// Pass 1: Forward declarations for structs, custom lists, and maps.
	for _, sig := range structOrder {
		cType := mangleStruct(sig.Name)
		fmt.Fprintf(out, "typedef struct %s %s;\n", cType, cType)
	}
	for _, col := range collections {
		fmt.Fprintf(out, "typedef struct rt_list_%s rt_list_%s;\n", col.name, col.name)
	}
	for _, m := range maps {
		fmt.Fprintf(out, "typedef struct rt_map_%s rt_map_%s;\n", m.name, m.name)
	}

	// Pass 2: Clone function prototypes and RT_CLONE_VALUE_* macros.
	for _, sig := range structOrder {
		cType := mangleStruct(sig.Name)
		rtName := fmt.Sprintf("struct_%d_%s", len(sig.Name), sig.Name)
		fmt.Fprintf(out, "static RT_UNUSED %s %s_clone(rt_arena* arena, %s value);\n", cType, cType, cType)
		fmt.Fprintf(out, "#define RT_CLONE_VALUE_%s(ARENA, VALUE) %s_clone((ARENA), (VALUE))\n", rtName, cType)
	}
	for _, p := range pointers {
		cType, err := emitType(p)
		if err != nil {
			return err
		}
		rtName, err := elemRuntimeName(p)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "static RT_UNUSED %s rt_%s_clone(rt_arena* arena, %s value);\n", cType, rtName, cType)
		fmt.Fprintf(out, "#define RT_CLONE_VALUE_%s(ARENA, VALUE) rt_%s_clone((ARENA), (VALUE))\n", rtName, rtName)
	}
	for _, m := range maps {
		cType := "rt_map_" + m.name + "*"
		fmt.Fprintf(out, "static RT_UNUSED %s rt_map_%s_clone(rt_arena* arena, %s value);\n", cType, m.name, cType)
		fmt.Fprintf(out, "#define RT_CLONE_VALUE_map_%s(ARENA, VALUE) rt_map_%s_clone((ARENA), (VALUE))\n", m.name, m.name)
	}
	for _, col := range collections {
		if strings.HasPrefix(col.name, "struct_") || strings.HasPrefix(col.name, "ptr_") || strings.HasPrefix(col.name, "map_") || col.name == "error" {
			continue
		}
		fmt.Fprintf(out, "#define RT_CLONE_VALUE_%s(ARENA, VALUE) %s((ARENA), (VALUE))\n", col.name, col.cloneFn)
	}

	// Pass 3: Struct definitions, RT_DEFINE_COLLECTIONS, and RT_DEFINE_MAP in topological order.
	emittedStructs := map[string]bool{}
	emittedCols := map[string]bool{}
	emittedMaps := map[string]bool{}

	var emitStructDef func(name string) error
	var emitColDef func(name string) error
	var emitMapDef func(name string) error
	var ensureValueTypeComplete func(typ ast.Type) error

	ensureValueTypeComplete = func(typ ast.Type) error {
		switch t := typ.(type) {
		case *ast.StructType:
			return emitStructDef(t.TargetName())
		case *ast.ArrayType:
			elemName, err := elemRuntimeName(t.Elem)
			if err != nil {
				return err
			}
			return emitColDef(elemName)
		case *ast.SliceType:
			elemName, err := elemRuntimeName(t.Elem)
			if err != nil {
				return err
			}
			return emitColDef(elemName)
		case *ast.ListType:
			elemName, err := elemRuntimeName(t.Elem)
			if err != nil {
				return err
			}
			return emitColDef(elemName)
		case *ast.MapType:
			mName, err := mapRuntimeName(t)
			if err != nil {
				return err
			}
			return emitMapDef(mName)
		default:
			return nil
		}
	}

	emitStructDef = func(name string) error {
		if emittedStructs[name] {
			return nil
		}
		sig, ok := program.Structs[name]
		if !ok {
			return nil
		}
		emittedStructs[name] = true
		for _, f := range sig.Fields {
			switch ft := f.Type.(type) {
			case *ast.StructType:
				if err := emitStructDef(ft.TargetName()); err != nil {
					return err
				}
			case *ast.ArrayType:
				elemName, err := elemRuntimeName(ft.Elem)
				if err != nil {
					return err
				}
				if err := emitColDef(elemName); err != nil {
					return err
				}
			case *ast.SliceType:
				elemName, err := elemRuntimeName(ft.Elem)
				if err != nil {
					return err
				}
				if err := emitColDef(elemName); err != nil {
					return err
				}
			}
		}
		cType := mangleStruct(sig.Name)
		fmt.Fprintf(out, "struct %s {\n", cType)
		if len(sig.Fields) == 0 {
			fmt.Fprintln(out, "    uint8_t _empty;")
		} else {
			for _, f := range sig.Fields {
				fCType, err := emitType(f.Type)
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "    %s %s;\n", fCType, mangleIdent(f.Name))
			}
		}
		fmt.Fprintln(out, "};")
		return nil
	}

	emitColDef = func(name string) error {
		if emittedCols[name] {
			return nil
		}
		col, ok := colByName[name]
		if !ok {
			return nil
		}
		emittedCols[name] = true
		for _, sig := range structOrder {
			rtName := fmt.Sprintf("struct_%d_%s", len(sig.Name), sig.Name)
			if rtName == name {
				if err := emitStructDef(sig.Name); err != nil {
					return err
				}
			}
		}
		fmt.Fprintf(out, "RT_DEFINE_COLLECTIONS(%s, %s)\n", col.name, col.cType)
		return nil
	}

	emitMapDef = func(name string) error {
		if emittedMaps[name] {
			return nil
		}
		m, ok := mapByName[name]
		if !ok {
			return nil
		}
		emittedMaps[name] = true
		if err := ensureValueTypeComplete(m.typ.Key); err != nil {
			return err
		}
		if err := ensureValueTypeComplete(m.typ.Value); err != nil {
			return err
		}
		fmt.Fprintf(out, "RT_DEFINE_MAP(%s, %s, %s, %s, %s, %s, %s, %s)\n",
			m.name, m.keyName, m.keyCType, m.valName, m.valCType, m.eqFn, m.hashFn, m.zeroVal)
		return nil
	}

	for _, sig := range structOrder {
		if err := emitStructDef(sig.Name); err != nil {
			return err
		}
	}
	for _, col := range collections {
		if err := emitColDef(col.name); err != nil {
			return err
		}
	}
	for _, m := range maps {
		if err := emitMapDef(m.name); err != nil {
			return err
		}
	}

	// Pass 4: RT_DEFINE_MAP_LISTS for all maps.
	for _, m := range maps {
		fmt.Fprintf(out, "RT_DEFINE_MAP_LISTS(%s, %s, %s, %s, %s)\n",
			m.name, m.keyName, m.keyCType, m.valName, m.valCType)
	}

	// Pass 5: Clone function bodies for structs and pointers.
	for _, sig := range structOrder {
		cType := mangleStruct(sig.Name)
		fmt.Fprintf(out, "static RT_UNUSED %s %s_clone(rt_arena* arena, %s value) {\n", cType, cType, cType)
		fmt.Fprintln(out, "    (void)arena;")
		fmt.Fprintln(out, "    (void)value;")
		if len(sig.Fields) == 0 {
			fmt.Fprintf(out, "    return RT_LIT(%s, 0);\n", cType)
		} else {
			clonedFields := make([]string, 0, len(sig.Fields))
			for _, f := range sig.Fields {
				cf, err := emitCloneValueExpr(f.Type, "arena", "value."+mangleIdent(f.Name))
				if err != nil {
					return err
				}
				clonedFields = append(clonedFields, cf)
			}
			fmt.Fprintf(out, "    return RT_LIT(%s, %s);\n", cType, strings.Join(clonedFields, ", "))
		}
		fmt.Fprintln(out, "}")
	}
	for _, p := range pointers {
		cType, err := emitType(p)
		if err != nil {
			return err
		}
		elemCType, err := emitType(p.Elem)
		if err != nil {
			return err
		}
		rtName, err := elemRuntimeName(p)
		if err != nil {
			return err
		}
		clonedElem, err := emitCloneValueExpr(p.Elem, "arena", "*value")
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "static RT_UNUSED %s rt_%s_clone(rt_arena* arena, %s value) {\n", cType, rtName, cType)
		fmt.Fprintln(out, "    if (value == NULL) {")
		fmt.Fprintln(out, "        return NULL;")
		fmt.Fprintln(out, "    }")
		fmt.Fprintf(out, "    %s copy = (%s)rt_arena_alloc(arena, sizeof(%s));\n", cType, cType, elemCType)
		fmt.Fprintf(out, "    *copy = %s;\n", clonedElem)
		fmt.Fprintln(out, "    return copy;")
		fmt.Fprintln(out, "}")
	}
	fmt.Fprintln(out)
	return nil
}

func mangleStruct(name string) string {
	return fmt.Sprintf("trux_s_%d_%s", len(name), name)
}

func mangleFunc(name string) string {
	return fmt.Sprintf("trux_f_%d_%s", len(name), name)
}

func mangleKernel(name string) string {
	return fmt.Sprintf("trux_k_%d_%s", len(name), name)
}

func mangleIdent(name string) string {
	return fmt.Sprintf("trux_v_%d_%s", len(name), name)
}

func stmtPos(stmt ir.Stmt) token.Position {
	switch stmt := stmt.(type) {
	case *ir.LetStmt:
		return stmt.Pos
	case *ir.ReturnStmt:
		return stmt.Pos
	case *ir.AssignStmt:
		return stmt.Pos
	case *ir.IndexAssignStmt:
		return stmt.Pos
	case *ir.FieldAssignStmt:
		return stmt.Pos
	case *ir.DerefAssignStmt:
		return stmt.Pos
	case *ir.DeleteStmt:
		return stmt.Pos
	case *ir.IfStmt:
		return stmt.Pos
	case *ir.ForStmt:
		return stmt.Pos
	case *ir.BreakStmt:
		return stmt.Pos
	case *ir.ContinueStmt:
		return stmt.Pos
	case *ir.PrintStmt:
		return stmt.Pos
	case *ir.AssertStmt:
		return stmt.Pos
	case *ir.AppendStmt:
		return stmt.Pos
	case *ir.WriteFileStmt:
		return stmt.Pos
	case *ir.WriteCSVStmt:
		return stmt.Pos
	case *ir.WritePPMStmt:
		return stmt.Pos
	case *ir.GPUCopyStmt:
		return stmt.Pos
	case *ir.GPUFreeStmt:
		return stmt.Pos
	case *ir.GPUSyncStmt:
		return stmt.Pos
	case *ir.GPULaunchStmt:
		return stmt.Pos
	case *ir.TimeSleepStmt:
		return stmt.Pos
	case *ir.HTTPServeStmt:
		return stmt.Pos
	case *ir.HTTPListenStmt:
		return stmt.Pos
	case *ir.HTTPUseStmt:
		return stmt.Pos
	case *ir.HTTPRouteStmt:
		return stmt.Pos
	case *ir.HTTPNotFoundStmt:
		return stmt.Pos
	case *ir.HTTPSetStmt:
		return stmt.Pos
	case *ir.HTTPSetHeaderStmt:
		return stmt.Pos
	case *ir.HTTPTextStmt:
		return stmt.Pos
	case *ir.HTTPJSONStmt:
		return stmt.Pos
	case *ir.HTTPStatusStmt:
		return stmt.Pos
	case *ir.HTTPRespondStmt:
		return stmt.Pos
	case *ir.ExprStmt:
		return stmt.Pos
	default:
		return token.Position{}
	}
}

func emitLine(out *bytes.Buffer, indent string, pos token.Position, sourceRoot string) {
	if pos.Line <= 0 || pos.File == "" {
		return
	}
	fmt.Fprintf(out, "%s#line %d %s\n", indent, pos.Line, cStringLiteral(sourcePath(pos.File, sourceRoot)))
}

func sourcePath(path string, sourceRoot string) string {
	if path == "" || sourceRoot == "" {
		return path
	}
	rel, err := filepath.Rel(sourceRoot, path)
	if err != nil || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return path
	}
	return filepath.ToSlash(rel)
}

func checkedArithmeticHelper(operator string) (string, bool) {
	switch operator {
	case "+":
		return "rt_add_i64", true
	case "-":
		return "rt_sub_i64", true
	case "*":
		return "rt_mul_i64", true
	case "/":
		return "rt_div_i64", true
	case "%":
		return "rt_rem_i64", true
	default:
		return "", false
	}
}

func gpuElemName(typ ast.Type) (string, error) {
	switch {
	case ast.TypeEqual(typ, ast.IntType):
		return "int", nil
	case ast.TypeEqual(typ, ast.FloatType):
		return "float", nil
	default:
		return "", fmt.Errorf("unsupported gpu buffer element type %s", typ)
	}
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

func tupleTypeName(typ *ast.TupleType) (string, error) {
	parts := make([]string, len(typ.Types))
	for i, elem := range typ.Types {
		name, err := elemRuntimeName(elem)
		if err != nil {
			return "", err
		}
		parts[i] = name
	}
	return "trux_tuple_" + strings.Join(parts, "_"), nil
}

func collectTupleTypes(program *ir.Program) ([]*ast.TupleType, error) {
	seen := map[string]bool{}
	var tuples []*ast.TupleType
	note := func(typ ast.Type) error {
		if typ == nil {
			return nil
		}
		if tuple, ok := typ.(*ast.TupleType); ok {
			name, err := tupleTypeName(tuple)
			if err != nil {
				return err
			}
			if !seen[name] {
				seen[name] = true
				tuples = append(tuples, tuple)
			}
		}
		return nil
	}
	for _, typ := range program.UsedTypes {
		if err := note(typ); err != nil {
			return nil, err
		}
	}
	for _, fn := range program.Functions {
		if err := note(fn.ReturnType); err != nil {
			return nil, err
		}
		for _, p := range fn.Params {
			if err := note(p.Type); err != nil {
				return nil, err
			}
		}
	}
	return tuples, nil
}

func sqliteRuntimeFnName(kind semtypes.SQLiteCallKind) string {
	switch kind {
	case semtypes.SQLiteCallOpen:
		return "rt_sqlite_open"
	case semtypes.SQLiteCallClose:
		return "rt_sqlite_close"
	case semtypes.SQLiteCallExec:
		return "rt_sqlite_exec"
	case semtypes.SQLiteCallQuery:
		return "rt_sqlite_query"
	case semtypes.SQLiteCallLastInsertRowID:
		return "rt_sqlite_last_insert_rowid"
	case semtypes.SQLiteCallChanges:
		return "rt_sqlite_changes"
	case semtypes.SQLiteCallTotalChanges:
		return "rt_sqlite_total_changes"
	case semtypes.SQLiteCallBegin:
		return "rt_sqlite_begin"
	case semtypes.SQLiteCallCommit:
		return "rt_sqlite_commit"
	case semtypes.SQLiteCallRollback:
		return "rt_sqlite_rollback"
	case semtypes.SQLiteCallPrepare:
		return "rt_sqlite_prepare"
	case semtypes.SQLiteCallStep:
		return "rt_sqlite_step"
	case semtypes.SQLiteCallReset:
		return "rt_sqlite_reset"
	case semtypes.SQLiteCallClearBindings:
		return "rt_sqlite_clear_bindings"
	case semtypes.SQLiteCallFinalize:
		return "rt_sqlite_finalize"
	case semtypes.SQLiteCallBindInt:
		return "rt_sqlite_bind_int"
	case semtypes.SQLiteCallBindFloat:
		return "rt_sqlite_bind_float"
	case semtypes.SQLiteCallBindText:
		return "rt_sqlite_bind_text"
	case semtypes.SQLiteCallBindNull:
		return "rt_sqlite_bind_null"
	case semtypes.SQLiteCallBindIndex:
		return "rt_sqlite_bind_index"
	case semtypes.SQLiteCallColumnCount:
		return "rt_sqlite_column_count"
	case semtypes.SQLiteCallColumnName:
		return "rt_sqlite_column_name"
	case semtypes.SQLiteCallColumnIndex:
		return "rt_sqlite_column_index"
	case semtypes.SQLiteCallColumnType:
		return "rt_sqlite_column_type"
	case semtypes.SQLiteCallColumnInt:
		return "rt_sqlite_column_int"
	case semtypes.SQLiteCallColumnFloat:
		return "rt_sqlite_column_float"
	case semtypes.SQLiteCallColumnText:
		return "rt_sqlite_column_text"
	case semtypes.SQLiteCallColumnIsNull:
		return "rt_sqlite_column_is_null"
	default:
		panic("unknown sqlite call kind: " + string(kind))
	}
}
