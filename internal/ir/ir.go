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

type IndexAssignStmt struct {
	Target *IndexExpr
	Value  Expr
}

func (*IndexAssignStmt) stmtNode() {}

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

type AppendStmt struct {
	List     Expr
	Value    Expr
	ListType ast.Type
	ElemType ast.Type
}

func (*AppendStmt) stmtNode() {}

type WriteFileStmt struct {
	Path     Expr
	Contents Expr
}

func (*WriteFileStmt) stmtNode() {}

type WriteCSVStmt struct {
	Path    Expr
	Cells   Expr
	Columns Expr
}

func (*WriteCSVStmt) stmtNode() {}

type ExprStmt struct {
	Expr Expr
}

func (*ExprStmt) stmtNode() {}

type Expr interface {
	Type() ast.Type
	Ownership() types.Origin
	exprNode()
}

type IdentExpr struct {
	Name   string
	Typ    ast.Type
	Origin types.Origin
}

func (e *IdentExpr) Type() ast.Type { return e.Typ }

func (e *IdentExpr) Ownership() types.Origin { return e.Origin }

func (*IdentExpr) exprNode() {}

type IntLiteral struct {
	Value  string
	Typ    ast.Type
	Origin types.Origin
}

func (e *IntLiteral) Type() ast.Type { return e.Typ }

func (e *IntLiteral) Ownership() types.Origin { return e.Origin }

func (*IntLiteral) exprNode() {}

type FloatLiteral struct {
	Value  string
	Typ    ast.Type
	Origin types.Origin
}

func (e *FloatLiteral) Type() ast.Type { return e.Typ }

func (e *FloatLiteral) Ownership() types.Origin { return e.Origin }

func (*FloatLiteral) exprNode() {}

type StringLiteral struct {
	Value  string
	Typ    ast.Type
	Origin types.Origin
}

func (e *StringLiteral) Type() ast.Type { return e.Typ }

func (e *StringLiteral) Ownership() types.Origin { return e.Origin }

func (*StringLiteral) exprNode() {}

type BoolLiteral struct {
	Value  bool
	Typ    ast.Type
	Origin types.Origin
}

func (e *BoolLiteral) Type() ast.Type { return e.Typ }

func (e *BoolLiteral) Ownership() types.Origin { return e.Origin }

func (*BoolLiteral) exprNode() {}

type ArrayLiteral struct {
	Elements []Expr
	Typ      ast.Type
	Origin   types.Origin
}

func (e *ArrayLiteral) Type() ast.Type { return e.Typ }

func (e *ArrayLiteral) Ownership() types.Origin { return e.Origin }

func (*ArrayLiteral) exprNode() {}

type ListLiteral struct {
	Elements []Expr
	Typ      ast.Type
	Origin   types.Origin
}

func (e *ListLiteral) Type() ast.Type { return e.Typ }

func (e *ListLiteral) Ownership() types.Origin { return e.Origin }

func (*ListLiteral) exprNode() {}

type MakeExpr struct {
	Len    Expr
	Typ    ast.Type
	Origin types.Origin
}

func (e *MakeExpr) Type() ast.Type { return e.Typ }

func (e *MakeExpr) Ownership() types.Origin { return e.Origin }

func (*MakeExpr) exprNode() {}

type CallExpr struct {
	Callee     string
	ReturnType ast.Type
	Args       []Expr
	Origin     types.Origin
}

func (e *CallExpr) Type() ast.Type { return e.ReturnType }

func (e *CallExpr) Ownership() types.Origin { return e.Origin }

func (*CallExpr) exprNode() {}

type CloneExpr struct {
	Value  Expr
	Typ    ast.Type
	Origin types.Origin
}

func (e *CloneExpr) Type() ast.Type { return e.Typ }

func (e *CloneExpr) Ownership() types.Origin { return e.Origin }

func (*CloneExpr) exprNode() {}

type ReadLineExpr struct {
	Typ    ast.Type
	Origin types.Origin
}

func (e *ReadLineExpr) Type() ast.Type { return e.Typ }

func (e *ReadLineExpr) Ownership() types.Origin { return e.Origin }

func (*ReadLineExpr) exprNode() {}

type ReadIntExpr struct {
	Typ    ast.Type
	Origin types.Origin
}

func (e *ReadIntExpr) Type() ast.Type { return e.Typ }

func (e *ReadIntExpr) Ownership() types.Origin { return e.Origin }

func (*ReadIntExpr) exprNode() {}

type ReadFloatExpr struct {
	Typ    ast.Type
	Origin types.Origin
}

func (e *ReadFloatExpr) Type() ast.Type { return e.Typ }

func (e *ReadFloatExpr) Ownership() types.Origin { return e.Origin }

func (*ReadFloatExpr) exprNode() {}

type ReadBoolExpr struct {
	Typ    ast.Type
	Origin types.Origin
}

func (e *ReadBoolExpr) Type() ast.Type { return e.Typ }

func (e *ReadBoolExpr) Ownership() types.Origin { return e.Origin }

func (*ReadBoolExpr) exprNode() {}

type ReadFileExpr struct {
	Path   Expr
	Typ    ast.Type
	Origin types.Origin
}

func (e *ReadFileExpr) Type() ast.Type { return e.Typ }

func (e *ReadFileExpr) Ownership() types.Origin { return e.Origin }

func (*ReadFileExpr) exprNode() {}

type ReadCSVExpr struct {
	Path    Expr
	Columns Expr
	Typ     ast.Type
	Origin  types.Origin
}

func (e *ReadCSVExpr) Type() ast.Type { return e.Typ }

func (e *ReadCSVExpr) Ownership() types.Origin { return e.Origin }

func (*ReadCSVExpr) exprNode() {}

type BinaryExpr struct {
	Left     Expr
	Operator string
	Right    Expr
	Typ      ast.Type
	Origin   types.Origin
}

func (e *BinaryExpr) Type() ast.Type { return e.Typ }

func (e *BinaryExpr) Ownership() types.Origin { return e.Origin }

func (*BinaryExpr) exprNode() {}

type LenExpr struct {
	Value  Expr
	Typ    ast.Type
	Origin types.Origin
}

func (e *LenExpr) Type() ast.Type { return e.Typ }

func (e *LenExpr) Ownership() types.Origin { return e.Origin }

func (*LenExpr) exprNode() {}

type IndexExpr struct {
	Collection Expr
	Index      Expr
	Typ        ast.Type
	Origin     types.Origin
}

func (e *IndexExpr) Type() ast.Type { return e.Typ }

func (e *IndexExpr) Ownership() types.Origin { return e.Origin }

func (*IndexExpr) exprNode() {}

type SliceExpr struct {
	Collection Expr
	Start      Expr
	End        Expr
	Typ        ast.Type
	Origin     types.Origin
}

func (e *SliceExpr) Type() ast.Type { return e.Typ }

func (e *SliceExpr) Ownership() types.Origin { return e.Origin }

func (*SliceExpr) exprNode() {}

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
	case *ast.IndexAssignStmt:
		target, err := b.buildExpr(stmt.Target)
		if err != nil {
			return nil, err
		}
		value, err := b.buildExpr(stmt.Value)
		if err != nil {
			return nil, err
		}
		return &IndexAssignStmt{Target: target.(*IndexExpr), Value: value}, nil
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
			if appendSig, ok := b.info.AppendCalls[call]; ok {
				args, err := b.buildExprs(call.Args)
				if err != nil {
					return nil, err
				}
				return &AppendStmt{List: args[0], Value: args[1], ListType: appendSig.ListType, ElemType: appendSig.ElemType}, nil
			}
			if ioSig, ok := b.info.IOCalls[call]; ok {
				switch ioSig.Kind {
				case types.IOCallWriteFile:
					args, err := b.buildExprs(call.Args)
					if err != nil {
						return nil, err
					}
					return &WriteFileStmt{Path: args[0], Contents: args[1]}, nil
				case types.IOCallWriteCSV:
					args, err := b.buildExprs(call.Args)
					if err != nil {
						return nil, err
					}
					return &WriteCSVStmt{Path: args[0], Cells: args[1], Columns: args[2]}, nil
				}
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
	origin, ok := b.info.ExprOrigins[expr]
	if !ok {
		origin = types.OriginUnknown
	}

	switch expr := expr.(type) {
	case *ast.IntLiteral:
		return &IntLiteral{Value: expr.Value, Typ: typ, Origin: origin}, nil
	case *ast.FloatLiteral:
		return &FloatLiteral{Value: expr.Value, Typ: typ, Origin: origin}, nil
	case *ast.StringLiteral:
		return &StringLiteral{Value: expr.Value, Typ: typ, Origin: origin}, nil
	case *ast.BoolLiteral:
		return &BoolLiteral{Value: expr.Value, Typ: typ, Origin: origin}, nil
	case *ast.ArrayLiteral:
		elements, err := b.buildExprs(expr.Elements)
		if err != nil {
			return nil, err
		}
		return &ArrayLiteral{Elements: elements, Typ: typ, Origin: origin}, nil
	case *ast.ListLiteral:
		elements, err := b.buildExprs(expr.Elements)
		if err != nil {
			return nil, err
		}
		return &ListLiteral{Elements: elements, Typ: typ, Origin: origin}, nil
	case *ast.MakeExpr:
		length, err := b.buildExpr(expr.Len)
		if err != nil {
			return nil, err
		}
		return &MakeExpr{Len: length, Typ: typ, Origin: origin}, nil
	case *ast.IdentExpr:
		return &IdentExpr{Name: expr.Name, Typ: typ, Origin: origin}, nil
	case *ast.CallExpr:
		if ioSig, ok := b.info.IOCalls[expr]; ok {
			return b.buildIOExpr(expr, ioSig, typ, origin)
		}
		if _, ok := b.info.CloneCalls[expr]; ok {
			arg, err := b.buildExpr(expr.Args[0])
			if err != nil {
				return nil, err
			}
			return &CloneExpr{Value: arg, Typ: typ, Origin: origin}, nil
		}
		if _, ok := b.info.LenCalls[expr]; ok {
			arg, err := b.buildExpr(expr.Args[0])
			if err != nil {
				return nil, err
			}
			return &LenExpr{Value: arg, Typ: typ, Origin: origin}, nil
		}
		sig, ok := b.info.ResolvedCalls[expr]
		if !ok {
			return nil, fmt.Errorf("missing resolved call for %q", expr.Callee)
		}
		args, err := b.buildExprs(expr.Args)
		if err != nil {
			return nil, err
		}
		return &CallExpr{Callee: sig.Name, ReturnType: sig.ReturnType, Args: args, Origin: origin}, nil
	case *ast.BinaryExpr:
		left, err := b.buildExpr(expr.Left)
		if err != nil {
			return nil, err
		}
		right, err := b.buildExpr(expr.Right)
		if err != nil {
			return nil, err
		}
		return &BinaryExpr{Left: left, Operator: expr.Operator, Right: right, Typ: typ, Origin: origin}, nil
	case *ast.IndexExpr:
		collection, err := b.buildExpr(expr.Collection)
		if err != nil {
			return nil, err
		}
		index, err := b.buildExpr(expr.Index)
		if err != nil {
			return nil, err
		}
		return &IndexExpr{Collection: collection, Index: index, Typ: typ, Origin: origin}, nil
	case *ast.SliceExpr:
		collection, err := b.buildExpr(expr.Collection)
		if err != nil {
			return nil, err
		}
		var start Expr
		if expr.StartIndex != nil {
			start, err = b.buildExpr(expr.StartIndex)
			if err != nil {
				return nil, err
			}
		}
		var end Expr
		if expr.EndIndex != nil {
			end, err = b.buildExpr(expr.EndIndex)
			if err != nil {
				return nil, err
			}
		}
		return &SliceExpr{Collection: collection, Start: start, End: end, Typ: typ, Origin: origin}, nil
	default:
		return nil, fmt.Errorf("unsupported AST expression %T", expr)
	}
}

func (b builder) buildIOExpr(expr *ast.CallExpr, sig types.IOCallSig, typ ast.Type, origin types.Origin) (Expr, error) {
	switch sig.Kind {
	case types.IOCallReadLine:
		return &ReadLineExpr{Typ: typ, Origin: origin}, nil
	case types.IOCallReadInt:
		return &ReadIntExpr{Typ: typ, Origin: origin}, nil
	case types.IOCallReadFloat:
		return &ReadFloatExpr{Typ: typ, Origin: origin}, nil
	case types.IOCallReadBool:
		return &ReadBoolExpr{Typ: typ, Origin: origin}, nil
	case types.IOCallReadFile:
		path, err := b.buildExpr(expr.Args[0])
		if err != nil {
			return nil, err
		}
		return &ReadFileExpr{Path: path, Typ: typ, Origin: origin}, nil
	case types.IOCallReadCSV:
		path, err := b.buildExpr(expr.Args[0])
		if err != nil {
			return nil, err
		}
		columns, err := b.buildExpr(expr.Args[1])
		if err != nil {
			return nil, err
		}
		return &ReadCSVExpr{Path: path, Columns: columns, Typ: typ, Origin: origin}, nil
	default:
		return nil, fmt.Errorf("%s can only be used as a statement", sig.Kind)
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
