package ir

import (
	"fmt"
	"strings"

	"github.com/claudioscheer/trux/internal/ast"
	"github.com/claudioscheer/trux/internal/stdlib"
	"github.com/claudioscheer/trux/internal/token"
	"github.com/claudioscheer/trux/internal/types"
)

type Program struct {
	PackageName string
	Functions   []*Func
	Kernels     []*Func
	Structs     map[string]types.StructSig
	UsesHTTP    bool
	UsesSQLite  bool
	UsesJSON    bool
	UsesJWT     bool
	UsesEnv     bool
	UsedTypes   []ast.Type
}

type Func struct {
	Name       string
	Pos        token.Position
	Kernel     bool
	Injected   int
	Params     []Param
	ReturnType ast.Type
	Body       []Stmt
}

type Param struct {
	Name    string
	Type    ast.Type
	Mutable bool
}

type Stmt interface {
	stmtNode()
}

type LetBinding struct {
	Pos  token.Position
	Name string
	Type ast.Type
}

type LetStmt struct {
	Name     string
	Pos      token.Position
	Type     ast.Type
	Bindings []LetBinding
	Value    Expr
}

func (*LetStmt) stmtNode() {}

type ReturnStmt struct {
	Pos    token.Position
	Value  Expr
	Values []Expr
}

func (*ReturnStmt) stmtNode() {}

type AssignTarget struct {
	Pos  token.Position
	Name string
}

type AssignStmt struct {
	Name    string
	Pos     token.Position
	Targets []AssignTarget
	Value   Expr
}

func (*AssignStmt) stmtNode() {}

type IndexAssignStmt struct {
	Target *IndexExpr
	Pos    token.Position
	Value  Expr
}

func (*IndexAssignStmt) stmtNode() {}

type FieldAssignStmt struct {
	Target *FieldExpr
	Pos    token.Position
	Value  Expr
}

func (*FieldAssignStmt) stmtNode() {}

type DerefAssignStmt struct {
	Target Expr
	Pos    token.Position
	Value  Expr
}

func (*DerefAssignStmt) stmtNode() {}

type DeleteStmt struct {
	Map     Expr
	Key     Expr
	Pos     token.Position
	MapType ast.Type
	KeyType ast.Type
}

func (*DeleteStmt) stmtNode() {}

type IfStmt struct {
	Pos       token.Position
	Condition Expr
	Then      []Stmt
	Else      []Stmt
}

func (*IfStmt) stmtNode() {}

type ForStmt struct {
	Pos       token.Position
	Init      Stmt
	Condition Expr
	Post      Stmt
	Body      []Stmt
}

func (*ForStmt) stmtNode() {}

type BreakStmt struct {
	Pos token.Position
}

func (*BreakStmt) stmtNode() {}

type ContinueStmt struct {
	Pos token.Position
}

func (*ContinueStmt) stmtNode() {}

type PrintStmt struct {
	Args  []Expr
	Pos   token.Position
	Types []ast.Type
}

func (*PrintStmt) stmtNode() {}

type AssertStmt struct {
	Condition Expr
	Message   Expr
	Pos       token.Position
}

func (*AssertStmt) stmtNode() {}

type AppendStmt struct {
	List     Expr
	Pos      token.Position
	Value    Expr
	ListType ast.Type
	ElemType ast.Type
}

func (*AppendStmt) stmtNode() {}

type WriteFileStmt struct {
	Pos      token.Position
	Path     Expr
	Contents Expr
}

func (*WriteFileStmt) stmtNode() {}

type WriteCSVStmt struct {
	Pos     token.Position
	Path    Expr
	Cells   Expr
	Columns Expr
}

func (*WriteCSVStmt) stmtNode() {}

type WritePPMStmt struct {
	Pos    token.Position
	Path   Expr
	Pixels Expr
	Width  Expr
	Height Expr
}

func (*WritePPMStmt) stmtNode() {}

type GPUCopyStmt struct {
	Kind   types.GPUCallKind
	Pos    token.Position
	First  Expr
	Second Expr
}

func (*GPUCopyStmt) stmtNode() {}

type GPUFreeStmt struct {
	Pos    token.Position
	Buffer Expr
}

func (*GPUFreeStmt) stmtNode() {}

type GPUSyncStmt struct {
	Pos token.Position
}

func (*GPUSyncStmt) stmtNode() {}

type GPUMountStmt struct {
	Pos  token.Position
	Path Expr
}

func (*GPUMountStmt) stmtNode() {}

type GPUWorkspaceStmt struct {
	Pos   token.Position
	Kind  types.GPUCallKind
	Name  Expr
	Count Expr
}

func (*GPUWorkspaceStmt) stmtNode() {}

type GPUFinishStmt struct {
	Pos         token.Position
	Request     Expr
	HandlerName string
}

func (*GPUFinishStmt) stmtNode() {}

type GPULaunchStmt struct {
	Pos        token.Position
	KernelName string
	Axes       int
	Dims       []Expr
	Args       []Expr
}

func (*GPULaunchStmt) stmtNode() {}

type TimeSleepStmt struct {
	Pos    token.Position
	Millis Expr
}

func (*TimeSleepStmt) stmtNode() {}

type HTTPServeStmt struct {
	Pos     token.Position
	Host    Expr
	Port    Expr
	Workers Expr
}

func (*HTTPServeStmt) stmtNode() {}

type HTTPListenStmt struct {
	Pos  token.Position
	Port Expr
}

func (*HTTPListenStmt) stmtNode() {}

type HTTPUseStmt struct {
	Pos         token.Position
	Prefix      Expr
	HandlerName string
}

func (*HTTPUseStmt) stmtNode() {}

type HTTPRouteStmt struct {
	Pos         token.Position
	Method      Expr
	Path        Expr
	HandlerName string
}

func (*HTTPRouteStmt) stmtNode() {}

type HTTPNotFoundStmt struct {
	Pos         token.Position
	HandlerName string
}

func (*HTTPNotFoundStmt) stmtNode() {}

type HTTPSetStmt struct {
	Pos     token.Position
	Request Expr
	Key     Expr
	Val     Expr
}

func (*HTTPSetStmt) stmtNode() {}

type HTTPSetHeaderStmt struct {
	Pos     token.Position
	Request Expr
	Name    Expr
	Val     Expr
}

func (*HTTPSetHeaderStmt) stmtNode() {}

type HTTPTextStmt struct {
	Pos     token.Position
	Request Expr
	Status  Expr
	Body    Expr
}

func (*HTTPTextStmt) stmtNode() {}

type HTTPJSONStmt struct {
	Pos     token.Position
	Request Expr
	Status  Expr
	Body    Expr
}

func (*HTTPJSONStmt) stmtNode() {}

type HTTPStatusStmt struct {
	Pos     token.Position
	Request Expr
	Status  Expr
}

func (*HTTPStatusStmt) stmtNode() {}

type HTTPRespondStmt struct {
	Pos         token.Position
	Request     Expr
	Status      Expr
	ContentType Expr
	Body        Expr
}

func (*HTTPRespondStmt) stmtNode() {}

type ExprStmt struct {
	Pos  token.Position
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

type NilLiteral struct {
	Typ    ast.Type
	Origin types.Origin
}

func (e *NilLiteral) Type() ast.Type { return e.Typ }

func (e *NilLiteral) Ownership() types.Origin { return e.Origin }

func (*NilLiteral) exprNode() {}

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

type StructFieldInit struct {
	Name  string
	Value Expr
}

type StructLiteral struct {
	Fields []StructFieldInit
	Typ    ast.Type
	Origin types.Origin
}

func (e *StructLiteral) Type() ast.Type { return e.Typ }

func (e *StructLiteral) Ownership() types.Origin { return e.Origin }

func (*StructLiteral) exprNode() {}

type MapEntry struct {
	Key   Expr
	Value Expr
}

type MapLiteral struct {
	Entries []MapEntry
	Typ     ast.Type
	Origin  types.Origin
}

func (e *MapLiteral) Type() ast.Type { return e.Typ }

func (e *MapLiteral) Ownership() types.Origin { return e.Origin }

func (*MapLiteral) exprNode() {}

type NewExpr struct {
	ElemType ast.Type
	Typ      ast.Type
	Origin   types.Origin
}

func (e *NewExpr) Type() ast.Type { return e.Typ }

func (e *NewExpr) Ownership() types.Origin { return e.Origin }

func (*NewExpr) exprNode() {}

type FieldExpr struct {
	Object Expr
	Field  string
	Pos    token.Position
	Typ    ast.Type
	Origin types.Origin
}

func (e *FieldExpr) Type() ast.Type { return e.Typ }

func (e *FieldExpr) Ownership() types.Origin { return e.Origin }

func (*FieldExpr) exprNode() {}

type MapLookupExpr struct {
	Map     Expr
	Key     Expr
	MapType *ast.MapType
	Typ     ast.Type
	Origin  types.Origin
}

func (e *MapLookupExpr) Type() ast.Type { return e.Typ }

func (e *MapLookupExpr) Ownership() types.Origin { return e.Origin }

func (*MapLookupExpr) exprNode() {}

type KeysExpr struct {
	Map     Expr
	MapType *ast.MapType
	Typ     ast.Type
	Origin  types.Origin
}

func (e *KeysExpr) Type() ast.Type { return e.Typ }

func (e *KeysExpr) Ownership() types.Origin { return e.Origin }

func (*KeysExpr) exprNode() {}

type ValuesExpr struct {
	Map     Expr
	MapType *ast.MapType
	Typ     ast.Type
	Origin  types.Origin
}

func (e *ValuesExpr) Type() ast.Type { return e.Typ }

func (e *ValuesExpr) Ownership() types.Origin { return e.Origin }

func (*ValuesExpr) exprNode() {}

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

type ErrorsNewExpr struct {
	Message Expr
	Typ     ast.Type
	Origin  types.Origin
}

func (e *ErrorsNewExpr) Type() ast.Type { return e.Typ }

func (e *ErrorsNewExpr) Ownership() types.Origin { return e.Origin }

func (*ErrorsNewExpr) exprNode() {}

type ErrorsMessageExpr struct {
	Err    Expr
	Typ    ast.Type
	Origin types.Origin
}

func (e *ErrorsMessageExpr) Type() ast.Type { return e.Typ }

func (e *ErrorsMessageExpr) Ownership() types.Origin { return e.Origin }

func (*ErrorsMessageExpr) exprNode() {}

type ReadCSVExpr struct {
	Path    Expr
	Columns Expr
	Typ     ast.Type
	Origin  types.Origin
}

func (e *ReadCSVExpr) Type() ast.Type { return e.Typ }

func (e *ReadCSVExpr) Ownership() types.Origin { return e.Origin }

func (*ReadCSVExpr) exprNode() {}

type WriteFileExpr struct {
	Path     Expr
	Contents Expr
	Typ      ast.Type
	Origin   types.Origin
}

func (e *WriteFileExpr) Type() ast.Type { return e.Typ }

func (e *WriteFileExpr) Ownership() types.Origin { return e.Origin }

func (*WriteFileExpr) exprNode() {}

type CreateDirExpr struct {
	Path   Expr
	Typ    ast.Type
	Origin types.Origin
}

func (e *CreateDirExpr) Type() ast.Type { return e.Typ }

func (e *CreateDirExpr) Ownership() types.Origin { return e.Origin }

func (*CreateDirExpr) exprNode() {}

type RemoveExpr struct {
	Path   Expr
	Typ    ast.Type
	Origin types.Origin
}

func (e *RemoveExpr) Type() ast.Type { return e.Typ }

func (e *RemoveExpr) Ownership() types.Origin { return e.Origin }

func (*RemoveExpr) exprNode() {}

type ReadDirExpr struct {
	Path   Expr
	Typ    ast.Type
	Origin types.Origin
}

func (e *ReadDirExpr) Type() ast.Type { return e.Typ }

func (e *ReadDirExpr) Ownership() types.Origin { return e.Origin }

func (*ReadDirExpr) exprNode() {}

type EnvExpr struct {
	Typ    ast.Type
	Origin types.Origin
}

func (e *EnvExpr) Type() ast.Type { return e.Typ }

func (e *EnvExpr) Ownership() types.Origin { return e.Origin }

func (*EnvExpr) exprNode() {}

type PathQueryExpr struct {
	Kind   types.IOCallKind
	Path   Expr
	Typ    ast.Type
	Origin types.Origin
}

func (e *PathQueryExpr) Type() ast.Type { return e.Typ }

func (e *PathQueryExpr) Ownership() types.Origin { return e.Origin }

func (*PathQueryExpr) exprNode() {}

type WriteCSVExpr struct {
	Path    Expr
	Cells   Expr
	Columns Expr
	Typ     ast.Type
	Origin  types.Origin
}

func (e *WriteCSVExpr) Type() ast.Type { return e.Typ }

func (e *WriteCSVExpr) Ownership() types.Origin { return e.Origin }

func (*WriteCSVExpr) exprNode() {}

type WritePPMExpr struct {
	Path   Expr
	Pixels Expr
	Width  Expr
	Height Expr
	Typ    ast.Type
	Origin types.Origin
}

func (e *WritePPMExpr) Type() ast.Type { return e.Typ }

func (e *WritePPMExpr) Ownership() types.Origin { return e.Origin }

func (*WritePPMExpr) exprNode() {}

type ImageDimensionExpr struct {
	Kind   types.IOCallKind
	Path   Expr
	Typ    ast.Type
	Origin types.Origin
}

func (e *ImageDimensionExpr) Type() ast.Type { return e.Typ }

func (e *ImageDimensionExpr) Ownership() types.Origin { return e.Origin }

func (*ImageDimensionExpr) exprNode() {}

type SQLiteCallExpr struct {
	Kind   types.SQLiteCallKind
	Args   []Expr
	Typ    ast.Type
	Origin types.Origin
}

func (e *SQLiteCallExpr) Type() ast.Type { return e.Typ }

func (e *SQLiteCallExpr) Ownership() types.Origin { return e.Origin }

func (*SQLiteCallExpr) exprNode() {}

type JSONCallExpr struct {
	Kind   types.JSONCallKind
	Args   []Expr
	Typ    ast.Type
	Origin types.Origin
}

func (e *JSONCallExpr) Type() ast.Type { return e.Typ }

func (e *JSONCallExpr) Ownership() types.Origin { return e.Origin }

func (*JSONCallExpr) exprNode() {}

type JWTCallExpr struct {
	Kind   types.JWTCallKind
	Args   []Expr
	Typ    ast.Type
	Origin types.Origin
}

func (e *JWTCallExpr) Type() ast.Type { return e.Typ }

func (e *JWTCallExpr) Ownership() types.Origin { return e.Origin }

func (*JWTCallExpr) exprNode() {}

type ReadPPMExpr struct {
	Path   Expr
	Typ    ast.Type
	Origin types.Origin
}

func (e *ReadPPMExpr) Type() ast.Type { return e.Typ }

func (e *ReadPPMExpr) Ownership() types.Origin { return e.Origin }

func (*ReadPPMExpr) exprNode() {}

type GPUAllocExpr struct {
	Len    Expr
	Typ    ast.Type
	Origin types.Origin
}

func (e *GPUAllocExpr) Type() ast.Type { return e.Typ }

func (e *GPUAllocExpr) Ownership() types.Origin { return e.Origin }

func (*GPUAllocExpr) exprNode() {}

type GPUFromExpr struct {
	Host   Expr
	Typ    ast.Type
	Origin types.Origin
}

func (e *GPUFromExpr) Type() ast.Type { return e.Typ }

func (e *GPUFromExpr) Ownership() types.Origin { return e.Origin }

func (*GPUFromExpr) exprNode() {}

type GPUCoordExpr struct {
	Kind   types.GPUCallKind
	Typ    ast.Type
	Origin types.Origin
}

func (e *GPUCoordExpr) Type() ast.Type { return e.Typ }

func (e *GPUCoordExpr) Ownership() types.Origin { return e.Origin }

func (*GPUCoordExpr) exprNode() {}

type GPUNameExpr struct {
	Kind   types.GPUCallKind
	Name   Expr
	Axis   Expr
	Typ    ast.Type
	Origin types.Origin
}

func (e *GPUNameExpr) Type() ast.Type { return e.Typ }

func (e *GPUNameExpr) Ownership() types.Origin { return e.Origin }

func (*GPUNameExpr) exprNode() {}

type GPUExpExpr struct {
	Arg    Expr
	Typ    ast.Type
	Origin types.Origin
}

func (e *GPUExpExpr) Type() ast.Type { return e.Typ }

func (e *GPUExpExpr) Ownership() types.Origin { return e.Origin }

func (*GPUExpExpr) exprNode() {}

type TimeNowExpr struct {
	Kind   types.IOCallKind
	Typ    ast.Type
	Origin types.Origin
}

func (e *TimeNowExpr) Type() ast.Type { return e.Typ }

func (e *TimeNowExpr) Ownership() types.Origin { return e.Origin }

func (*TimeNowExpr) exprNode() {}

type HTTPRequestStringExpr struct {
	Kind    types.HTTPCallKind
	Request Expr
	Name    Expr
	Typ     ast.Type
	Origin  types.Origin
}

func (e *HTTPRequestStringExpr) Type() ast.Type { return e.Typ }

func (e *HTTPRequestStringExpr) Ownership() types.Origin { return e.Origin }

func (*HTTPRequestStringExpr) exprNode() {}

type UnaryExpr struct {
	Operator string
	Operand  Expr
	Pos      token.Position
	Typ      ast.Type
	Origin   types.Origin
}

func (e *UnaryExpr) Type() ast.Type { return e.Typ }

func (e *UnaryExpr) Ownership() types.Origin { return e.Origin }

func (*UnaryExpr) exprNode() {}

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
	builder := builder{info: info, used: &usedTypes{info: info, seen: map[string]bool{}}}
	out, err := builder.buildProgram(program)
	if err != nil {
		return nil, err
	}
	if out.UsesSQLite {
		builder.used.note(stdlib.SQLiteDBType)
		builder.used.note(stdlib.SQLiteStmtType)
		builder.used.note(stdlib.SQLiteRowListType)
		builder.used.note(&ast.TupleType{Types: []ast.Type{stdlib.SQLiteDBType, ast.ErrorType}})
		builder.used.note(&ast.TupleType{Types: []ast.Type{stdlib.SQLiteStmtType, ast.ErrorType}})
		builder.used.note(&ast.TupleType{Types: []ast.Type{ast.BoolType, ast.ErrorType}})
		builder.used.note(&ast.TupleType{Types: []ast.Type{stdlib.SQLiteRowListType, ast.ErrorType}})
	}
	if out.UsesJSON {
		builder.used.note(stdlib.JSONDocType)
		builder.used.note(stdlib.JSONValType)
		builder.used.note(stdlib.JSONMutDocType)
		builder.used.note(stdlib.JSONMutValType)
		builder.used.note(stdlib.JSONMapType)
		builder.used.note(&ast.TupleType{Types: []ast.Type{stdlib.JSONDocType, ast.ErrorType}})
		builder.used.note(&ast.TupleType{Types: []ast.Type{stdlib.JSONMapType, ast.ErrorType}})
	}
	if out.UsesJWT {
		builder.used.note(stdlib.JWTClaimsType)
		builder.used.note(&ast.TupleType{Types: []ast.Type{ast.StringType, ast.ErrorType}})
		builder.used.note(&ast.TupleType{Types: []ast.Type{stdlib.JWTClaimsType, ast.ErrorType}})
	}
	out.UsedTypes = builder.used.list
	return out, nil
}

type usedTypes struct {
	info *types.Info
	list []ast.Type
	seen map[string]bool
}

func canonicalTypeKey(typ ast.Type) string {
	switch t := typ.(type) {
	case ast.ScalarType:
		return string(t)
	case *ast.StructType:
		return "struct:" + t.TargetName()
	case *ast.PointerType:
		return "*" + canonicalTypeKey(t.Elem)
	case *ast.ArrayType:
		return fmt.Sprintf("[%d]%s", t.Length, canonicalTypeKey(t.Elem))
	case *ast.SliceType:
		return "[]" + canonicalTypeKey(t.Elem)
	case *ast.ListType:
		return "list[" + canonicalTypeKey(t.Elem) + "]"
	case *ast.MapType:
		return "map[" + canonicalTypeKey(t.Key) + "]" + canonicalTypeKey(t.Value)
	case *ast.GPUBufferType:
		return "gpu.Buffer[" + canonicalTypeKey(t.Elem) + "]"
	case *ast.TupleType:
		var parts []string
		for _, elem := range t.Types {
			parts = append(parts, canonicalTypeKey(elem))
		}
		return "(" + strings.Join(parts, ", ") + ")"
	default:
		return typ.String()
	}
}

func (u *usedTypes) note(typ ast.Type) {
	if u == nil || typ == nil {
		return
	}
	key := canonicalTypeKey(typ)
	if u.seen[key] {
		return
	}
	u.seen[key] = true
	switch t := typ.(type) {
	case *ast.ArrayType:
		u.note(t.Elem)
	case *ast.SliceType:
		u.note(t.Elem)
	case *ast.ListType:
		u.note(t.Elem)
	case *ast.MapType:
		u.note(t.Key)
		u.note(t.Value)
	case *ast.PointerType:
		u.note(t.Elem)
	case *ast.GPUBufferType:
		u.note(t.Elem)
	case *ast.TupleType:
		for _, elem := range t.Types {
			u.note(elem)
		}
	case *ast.StructType:
		if u.info != nil {
			if sig, ok := u.info.Structs[t.TargetName()]; ok {
				for _, field := range sig.Fields {
					u.note(field.Type)
				}
			}
		}
	}
	u.list = append(u.list, typ)
}

type builder struct {
	info *types.Info
	used *usedTypes
}

func (b builder) buildProgram(program *ast.Program) (*Program, error) {
	structs := make(map[string]types.StructSig, len(b.info.Structs))
	for k, v := range b.info.Structs {
		structs[k] = v
	}
	out := &Program{PackageName: program.PackageName, Structs: structs, UsesHTTP: len(b.info.HTTPCalls) > 0, UsesSQLite: b.info.UsesSQLite, UsesJSON: b.info.UsesJSON, UsesJWT: b.info.UsesJWT, UsesEnv: b.info.UsesEnv}
	for _, fn := range program.Functions {
		irFn, err := b.buildFunc(fn)
		if err != nil {
			return nil, err
		}
		if fn.Kernel {
			out.Kernels = append(out.Kernels, irFn)
		} else {
			out.Functions = append(out.Functions, irFn)
		}
	}

	return out, nil
}

func (b builder) buildFunc(fn *ast.FuncDecl) (*Func, error) {
	b.used.note(fn.ReturnType)
	out := &Func{
		Name:       fn.Name,
		Pos:        fn.Pos,
		Kernel:     fn.Kernel,
		Injected:   b.info.KernelAxes[fn.Name],
		ReturnType: fn.ReturnType,
	}
	for _, param := range fn.Params {
		b.used.note(param.Type)
		out.Params = append(out.Params, Param{Name: param.Name, Type: param.Type, Mutable: param.Mutable})
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
		for _, binding := range stmt.Bindings {
			if binding.Type != nil {
				b.used.note(binding.Type)
			}
		}
		if stmt.Type != nil {
			b.used.note(stmt.Type)
		}
		value, err := b.buildExpr(stmt.Value)
		if err != nil {
			return nil, err
		}
		var bindings []LetBinding
		for _, binding := range stmt.Bindings {
			bindings = append(bindings, LetBinding{
				Pos:  binding.Pos,
				Name: binding.Name,
				Type: binding.Type,
			})
		}
		return &LetStmt{
			Name:     stmt.Name,
			Pos:      stmt.Start,
			Type:     stmt.Type,
			Bindings: bindings,
			Value:    value,
		}, nil
	case *ast.ReturnStmt:
		astValues := stmt.Values
		if len(astValues) == 0 && stmt.Value != nil {
			astValues = []ast.Expression{stmt.Value}
		}
		values, err := b.buildExprs(astValues)
		if err != nil {
			return nil, err
		}
		var first Expr
		if len(values) > 0 {
			first = values[0]
		}
		return &ReturnStmt{Pos: stmt.Pos(), Value: first, Values: values}, nil
	case *ast.AssignStmt:
		value, err := b.buildExpr(stmt.Value)
		if err != nil {
			return nil, err
		}
		var targets []AssignTarget
		for _, t := range stmt.Targets {
			targets = append(targets, AssignTarget{
				Pos:  t.Pos,
				Name: t.Name,
			})
		}
		return &AssignStmt{
			Name:    stmt.Name,
			Pos:     stmt.Start,
			Targets: targets,
			Value:   value,
		}, nil
	case *ast.IndexAssignStmt:
		target, err := b.buildExpr(stmt.Target)
		if err != nil {
			return nil, err
		}
		value, err := b.buildExpr(stmt.Value)
		if err != nil {
			return nil, err
		}
		return &IndexAssignStmt{Target: target.(*IndexExpr), Pos: stmt.Pos(), Value: value}, nil
	case *ast.FieldAssignStmt:
		target, err := b.buildExpr(stmt.Target)
		if err != nil {
			return nil, err
		}
		value, err := b.buildExpr(stmt.Value)
		if err != nil {
			return nil, err
		}
		return &FieldAssignStmt{Target: target.(*FieldExpr), Pos: stmt.Pos(), Value: value}, nil
	case *ast.DerefAssignStmt:
		target, err := b.buildExpr(stmt.Target.Operand)
		if err != nil {
			return nil, err
		}
		value, err := b.buildExpr(stmt.Value)
		if err != nil {
			return nil, err
		}
		return &DerefAssignStmt{Target: target, Pos: stmt.Pos(), Value: value}, nil
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
		return &IfStmt{Pos: stmt.Pos(), Condition: condition, Then: thenStmts, Else: elseStmts}, nil
	case *ast.ForStmt:
		var init Stmt
		if stmt.Init != nil {
			var err error
			init, err = b.buildStmt(stmt.Init)
			if err != nil {
				return nil, err
			}
		}
		var condition Expr
		if stmt.Condition != nil {
			var err error
			condition, err = b.buildExpr(stmt.Condition)
			if err != nil {
				return nil, err
			}
		}
		body, err := b.buildStmts(stmt.Body.Statements)
		if err != nil {
			return nil, err
		}
		var post Stmt
		if stmt.Post != nil {
			post, err = b.buildStmt(stmt.Post)
			if err != nil {
				return nil, err
			}
		}
		return &ForStmt{Pos: stmt.Pos(), Init: init, Condition: condition, Post: post, Body: body}, nil
	case *ast.BreakStmt:
		return &BreakStmt{Pos: stmt.Pos()}, nil
	case *ast.ContinueStmt:
		return &ContinueStmt{Pos: stmt.Pos()}, nil
	case *ast.ExprStmt:
		if call, ok := stmt.Expr.(*ast.CallExpr); ok {
			if _, ok := b.info.AssertCalls[call]; ok {
				args, err := b.buildExprs(call.Args)
				if err != nil {
					return nil, err
				}
				return &AssertStmt{Condition: args[0], Message: args[1], Pos: call.Start}, nil
			}
			if printTypes, ok := b.info.PrintCalls[call]; ok {
				args, err := b.buildExprs(call.Args)
				if err != nil {
					return nil, err
				}
				return &PrintStmt{Args: args, Pos: stmt.Pos(), Types: printTypes}, nil
			}
			if appendSig, ok := b.info.AppendCalls[call]; ok {
				args, err := b.buildExprs(call.Args)
				if err != nil {
					return nil, err
				}
				return &AppendStmt{List: args[0], Pos: stmt.Pos(), Value: args[1], ListType: appendSig.ListType, ElemType: appendSig.ElemType}, nil
			}
			if deleteSig, ok := b.info.DeleteCalls[call]; ok {
				args, err := b.buildExprs(call.Args)
				if err != nil {
					return nil, err
				}
				return &DeleteStmt{Map: args[0], Key: args[1], Pos: stmt.Pos(), MapType: deleteSig, KeyType: deleteSig.Key}, nil
			}
			if ioSig, ok := b.info.IOCalls[call]; ok {
				switch ioSig.Kind {
				case types.IOCallWriteFile:
					args, err := b.buildExprs(call.Args)
					if err != nil {
						return nil, err
					}
					return &WriteFileStmt{Pos: stmt.Pos(), Path: args[0], Contents: args[1]}, nil
				case types.IOCallWriteCSV:
					args, err := b.buildExprs(call.Args)
					if err != nil {
						return nil, err
					}
					return &WriteCSVStmt{Pos: stmt.Pos(), Path: args[0], Cells: args[1], Columns: args[2]}, nil
				case types.IOCallWritePPM:
					args, err := b.buildExprs(call.Args)
					if err != nil {
						return nil, err
					}
					return &WritePPMStmt{Pos: stmt.Pos(), Path: args[0], Pixels: args[1], Width: args[2], Height: args[3]}, nil
				case types.IOCallTimeSleepMillis:
					args, err := b.buildExprs(call.Args)
					if err != nil {
						return nil, err
					}
					return &TimeSleepStmt{Pos: stmt.Pos(), Millis: args[0]}, nil
				}
			}
			if httpSig, ok := b.info.HTTPCalls[call]; ok {
				switch httpSig.Kind {
				case types.HTTPCallListen:
					arg, err := b.buildExpr(call.Args[0])
					if err != nil {
						return nil, err
					}
					return &HTTPListenStmt{Pos: stmt.Pos(), Port: arg}, nil
				case types.HTTPCallServe:
					args, err := b.buildExprs(call.Args)
					if err != nil {
						return nil, err
					}
					return &HTTPServeStmt{Pos: stmt.Pos(), Host: args[0], Port: args[1], Workers: args[2]}, nil
				case types.HTTPCallUse:
					return &HTTPUseStmt{Pos: stmt.Pos(), HandlerName: httpSig.HandlerName}, nil
				case types.HTTPCallUseAt:
					prefix, err := b.buildExpr(call.Args[0])
					if err != nil {
						return nil, err
					}
					return &HTTPUseStmt{Pos: stmt.Pos(), Prefix: prefix, HandlerName: httpSig.HandlerName}, nil
				case types.HTTPCallGet, types.HTTPCallPost, types.HTTPCallPut, types.HTTPCallDelete, types.HTTPCallPatch:
					path, err := b.buildExpr(call.Args[0])
					if err != nil {
						return nil, err
					}
					method := &StringLiteral{Value: strings.ToUpper(string(httpSig.Kind))}
					return &HTTPRouteStmt{Pos: stmt.Pos(), Method: method, Path: path, HandlerName: httpSig.HandlerName}, nil
				case types.HTTPCallRoute:
					method, err := b.buildExpr(call.Args[0])
					if err != nil {
						return nil, err
					}
					path, err := b.buildExpr(call.Args[1])
					if err != nil {
						return nil, err
					}
					return &HTTPRouteStmt{Pos: stmt.Pos(), Method: method, Path: path, HandlerName: httpSig.HandlerName}, nil
				case types.HTTPCallNotFound:
					return &HTTPNotFoundStmt{Pos: stmt.Pos(), HandlerName: httpSig.HandlerName}, nil
				case types.HTTPCallSet:
					args, err := b.buildExprs(call.Args)
					if err != nil {
						return nil, err
					}
					return &HTTPSetStmt{Pos: stmt.Pos(), Request: args[0], Key: args[1], Val: args[2]}, nil
				case types.HTTPCallSetHeader:
					args, err := b.buildExprs(call.Args)
					if err != nil {
						return nil, err
					}
					return &HTTPSetHeaderStmt{Pos: stmt.Pos(), Request: args[0], Name: args[1], Val: args[2]}, nil
				case types.HTTPCallText:
					args, err := b.buildExprs(call.Args)
					if err != nil {
						return nil, err
					}
					return &HTTPTextStmt{Pos: stmt.Pos(), Request: args[0], Status: args[1], Body: args[2]}, nil
				case types.HTTPCallJSON:
					args, err := b.buildExprs(call.Args)
					if err != nil {
						return nil, err
					}
					return &HTTPJSONStmt{Pos: stmt.Pos(), Request: args[0], Status: args[1], Body: args[2]}, nil
				case types.HTTPCallStatus:
					args, err := b.buildExprs(call.Args)
					if err != nil {
						return nil, err
					}
					return &HTTPStatusStmt{Pos: stmt.Pos(), Request: args[0], Status: args[1]}, nil
				case types.HTTPCallRespond:
					args, err := b.buildExprs(call.Args)
					if err != nil {
						return nil, err
					}
					return &HTTPRespondStmt{Pos: stmt.Pos(), Request: args[0], Status: args[1], ContentType: args[2], Body: args[3]}, nil
				}
			}
			if gpuSig, ok := b.info.GPUCalls[call]; ok {
				if gpuSig.Kind == types.GPUCallLaunch || gpuSig.Kind == types.GPUCallLaunch1D || gpuSig.Kind == types.GPUCallLaunch2D {
					dimCount := 6
					axes := 0
					switch gpuSig.Kind {
					case types.GPUCallLaunch1D:
						dimCount = 1
						axes = 1
					case types.GPUCallLaunch2D:
						dimCount = 2
						axes = 2
					}
					dims, err := b.buildExprs(call.Args[1 : 1+dimCount])
					if err != nil {
						return nil, err
					}
					kernelArgs, err := b.buildExprs(call.Args[1+dimCount:])
					if err != nil {
						return nil, err
					}
					return &GPULaunchStmt{Pos: stmt.Pos(), KernelName: gpuSig.KernelName, Axes: axes, Dims: dims, Args: kernelArgs}, nil
				}
				if gpuSig.Kind == types.GPUCallFinish {
					request, err := b.buildExpr(call.Args[0])
					if err != nil {
						return nil, err
					}
					return &GPUFinishStmt{Pos: stmt.Pos(), Request: request, HandlerName: gpuSig.HandlerName}, nil
				}
				args, err := b.buildExprs(call.Args)
				if err != nil {
					return nil, err
				}
				switch gpuSig.Kind {
				case types.GPUCallCopyToDevice, types.GPUCallCopyToHost:
					return &GPUCopyStmt{Kind: gpuSig.Kind, Pos: stmt.Pos(), First: args[0], Second: args[1]}, nil
				case types.GPUCallFree:
					return &GPUFreeStmt{Pos: stmt.Pos(), Buffer: args[0]}, nil
				case types.GPUCallSync:
					return &GPUSyncStmt{Pos: stmt.Pos()}, nil
				case types.GPUCallMount:
					return &GPUMountStmt{Pos: stmt.Pos(), Path: args[0]}, nil
				case types.GPUCallWorkspaceFloat, types.GPUCallWorkspaceInt:
					return &GPUWorkspaceStmt{Pos: stmt.Pos(), Kind: gpuSig.Kind, Name: args[0], Count: args[1]}, nil
				}
			}
		}

		expr, err := b.buildExpr(stmt.Expr)
		if err != nil {
			return nil, err
		}
		return &ExprStmt{Pos: stmt.Pos(), Expr: expr}, nil
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
	b.used.note(typ)

	switch expr := expr.(type) {
	case *ast.IntLiteral:
		return &IntLiteral{Value: expr.Value, Typ: typ, Origin: origin}, nil
	case *ast.FloatLiteral:
		return &FloatLiteral{Value: expr.Value, Typ: typ, Origin: origin}, nil
	case *ast.StringLiteral:
		return &StringLiteral{Value: expr.Value, Typ: typ, Origin: origin}, nil
	case *ast.BoolLiteral:
		return &BoolLiteral{Value: expr.Value, Typ: typ, Origin: origin}, nil
	case *ast.NilLiteral:
		return &NilLiteral{Typ: typ, Origin: origin}, nil
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
	case *ast.StructLiteral:
		var fields []StructFieldInit
		for _, field := range expr.Fields {
			val, err := b.buildExpr(field.Value)
			if err != nil {
				return nil, err
			}
			fields = append(fields, StructFieldInit{Name: field.Name, Value: val})
		}
		return &StructLiteral{Fields: fields, Typ: typ, Origin: origin}, nil
	case *ast.MapLiteral:
		var entries []MapEntry
		for _, entry := range expr.Entries {
			k, err := b.buildExpr(entry.Key)
			if err != nil {
				return nil, err
			}
			v, err := b.buildExpr(entry.Value)
			if err != nil {
				return nil, err
			}
			entries = append(entries, MapEntry{Key: k, Value: v})
		}
		return &MapLiteral{Entries: entries, Typ: typ, Origin: origin}, nil
	case *ast.NewExpr:
		b.used.note(expr.ElemType)
		return &NewExpr{ElemType: expr.ElemType, Typ: typ, Origin: origin}, nil
	case *ast.MakeExpr:
		var length Expr
		if expr.Len != nil {
			var err error
			length, err = b.buildExpr(expr.Len)
			if err != nil {
				return nil, err
			}
		}
		return &MakeExpr{Len: length, Typ: typ, Origin: origin}, nil
	case *ast.IdentExpr:
		return &IdentExpr{Name: expr.Name, Typ: typ, Origin: origin}, nil
	case *ast.CallExpr:
		if gpuSig, ok := b.info.GPUCalls[expr]; ok {
			switch gpuSig.Kind {
			case types.GPUCallAlloc:
				length, err := b.buildExpr(expr.Args[0])
				if err != nil {
					return nil, err
				}
				return &GPUAllocExpr{Len: length, Typ: typ, Origin: origin}, nil
			case types.GPUCallFrom:
				host, err := b.buildExpr(expr.Args[0])
				if err != nil {
					return nil, err
				}
				return &GPUFromExpr{Host: host, Typ: typ, Origin: origin}, nil
			case types.GPUCallGlobalX, types.GPUCallGlobalY, types.GPUCallGlobalZ, types.GPUCallThreadX, types.GPUCallThreadY, types.GPUCallThreadZ, types.GPUCallBlockX, types.GPUCallBlockY, types.GPUCallBlockZ, types.GPUCallBlockDimX, types.GPUCallBlockDimY, types.GPUCallBlockDimZ:
				return &GPUCoordExpr{Kind: gpuSig.Kind, Typ: typ, Origin: origin}, nil
			case types.GPUCallExp:
				arg, err := b.buildExpr(expr.Args[0])
				if err != nil {
					return nil, err
				}
				return &GPUExpExpr{Arg: arg, Typ: typ, Origin: origin}, nil
			case types.GPUCallFloats, types.GPUCallInts, types.GPUCallBorrowFloat, types.GPUCallBorrowInt, types.GPUCallHostFloats, types.GPUCallHostInts, types.GPUCallScalar, types.GPUCallReady:
				name, err := b.buildExpr(expr.Args[0])
				if err != nil {
					return nil, err
				}
				return &GPUNameExpr{Kind: gpuSig.Kind, Name: name, Typ: typ, Origin: origin}, nil
			case types.GPUCallDim:
				name, err := b.buildExpr(expr.Args[0])
				if err != nil {
					return nil, err
				}
				axis, err := b.buildExpr(expr.Args[1])
				if err != nil {
					return nil, err
				}
				return &GPUNameExpr{Kind: gpuSig.Kind, Name: name, Axis: axis, Typ: typ, Origin: origin}, nil
			}
		}
		if ioSig, ok := b.info.IOCalls[expr]; ok {
			return b.buildIOExpr(expr, ioSig, typ, origin)
		}
		if sqliteSig, ok := b.info.SQLiteCalls[expr]; ok {
			args, err := b.buildExprs(expr.Args)
			if err != nil {
				return nil, err
			}
			return &SQLiteCallExpr{Kind: sqliteSig.Kind, Args: args, Typ: typ, Origin: origin}, nil
		}
		if jsonSig, ok := b.info.JSONCalls[expr]; ok {
			args, err := b.buildExprs(expr.Args)
			if err != nil {
				return nil, err
			}
			return &JSONCallExpr{Kind: jsonSig.Kind, Args: args, Typ: typ, Origin: origin}, nil
		}
		if jwtSig, ok := b.info.JWTCalls[expr]; ok {
			args, err := b.buildExprs(expr.Args)
			if err != nil {
				return nil, err
			}
			return &JWTCallExpr{Kind: jwtSig.Kind, Args: args, Typ: typ, Origin: origin}, nil
		}
		if httpSig, ok := b.info.HTTPCalls[expr]; ok {
			return b.buildHTTPExpr(expr, httpSig, typ, origin)
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
		if mt, ok := b.info.KeysCalls[expr]; ok {
			arg, err := b.buildExpr(expr.Args[0])
			if err != nil {
				return nil, err
			}
			return &KeysExpr{Map: arg, MapType: mt, Typ: typ, Origin: origin}, nil
		}
		if mt, ok := b.info.ValuesCalls[expr]; ok {
			arg, err := b.buildExpr(expr.Args[0])
			if err != nil {
				return nil, err
			}
			return &ValuesExpr{Map: arg, MapType: mt, Typ: typ, Origin: origin}, nil
		}
		sig, ok := b.info.ResolvedCalls[expr]
		if !ok {
			return nil, fmt.Errorf("missing resolved call for %q", expr.SourceName())
		}
		args, err := b.buildExprs(expr.Args)
		if err != nil {
			return nil, err
		}
		return &CallExpr{Callee: sig.Name, ReturnType: sig.ReturnType, Args: args, Origin: origin}, nil
	case *ast.UnaryExpr:
		operand, err := b.buildExpr(expr.Operand)
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Operator: expr.Operator, Operand: operand, Pos: expr.Start, Typ: typ, Origin: origin}, nil
	case *ast.FieldExpr:
		object, err := b.buildExpr(expr.Object)
		if err != nil {
			return nil, err
		}
		return &FieldExpr{Object: object, Field: expr.Field, Pos: expr.Start, Typ: typ, Origin: origin}, nil
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
		if mt, ok := b.info.MapLookups[expr]; ok {
			return &MapLookupExpr{Map: collection, Key: index, MapType: mt, Typ: typ, Origin: origin}, nil
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
	case types.IOCallImageWidth, types.IOCallImageHeight:
		path, err := b.buildExpr(expr.Args[0])
		if err != nil {
			return nil, err
		}
		return &ImageDimensionExpr{Kind: sig.Kind, Path: path, Typ: typ, Origin: origin}, nil
	case types.IOCallReadPPM:
		path, err := b.buildExpr(expr.Args[0])
		if err != nil {
			return nil, err
		}
		return &ReadPPMExpr{Path: path, Typ: typ, Origin: origin}, nil
	case types.IOCallTimeNowUnixMillis, types.IOCallTimeMonotonicNanos:
		return &TimeNowExpr{Kind: sig.Kind, Typ: typ, Origin: origin}, nil
	case types.IOCallErrorsNew:
		msg, err := b.buildExpr(expr.Args[0])
		if err != nil {
			return nil, err
		}
		return &ErrorsNewExpr{Message: msg, Typ: typ, Origin: origin}, nil
	case types.IOCallErrorsMessage:
		errExpr, err := b.buildExpr(expr.Args[0])
		if err != nil {
			return nil, err
		}
		return &ErrorsMessageExpr{Err: errExpr, Typ: typ, Origin: origin}, nil
	case types.IOCallWriteFile:
		args, err := b.buildExprs(expr.Args)
		if err != nil {
			return nil, err
		}
		return &WriteFileExpr{Path: args[0], Contents: args[1], Typ: typ, Origin: origin}, nil
	case types.IOCallCreateDir:
		path, err := b.buildExpr(expr.Args[0])
		if err != nil {
			return nil, err
		}
		return &CreateDirExpr{Path: path, Typ: typ, Origin: origin}, nil
	case types.IOCallRemove:
		path, err := b.buildExpr(expr.Args[0])
		if err != nil {
			return nil, err
		}
		return &RemoveExpr{Path: path, Typ: typ, Origin: origin}, nil
	case types.IOCallReadDir:
		path, err := b.buildExpr(expr.Args[0])
		if err != nil {
			return nil, err
		}
		return &ReadDirExpr{Path: path, Typ: typ, Origin: origin}, nil
	case types.IOCallEnv:
		return &EnvExpr{Typ: typ, Origin: origin}, nil
	case types.IOCallExists, types.IOCallIsFile, types.IOCallIsDir:
		path, err := b.buildExpr(expr.Args[0])
		if err != nil {
			return nil, err
		}
		return &PathQueryExpr{Kind: sig.Kind, Path: path, Typ: typ, Origin: origin}, nil
	case types.IOCallWriteCSV:
		args, err := b.buildExprs(expr.Args)
		if err != nil {
			return nil, err
		}
		return &WriteCSVExpr{Path: args[0], Cells: args[1], Columns: args[2], Typ: typ, Origin: origin}, nil
	case types.IOCallWritePPM:
		args, err := b.buildExprs(expr.Args)
		if err != nil {
			return nil, err
		}
		return &WritePPMExpr{Path: args[0], Pixels: args[1], Width: args[2], Height: args[3], Typ: typ, Origin: origin}, nil
	default:
		return nil, fmt.Errorf("%s can only be used as a statement", sig.Kind)
	}
}

func (b builder) buildHTTPExpr(expr *ast.CallExpr, sig types.HTTPCallSig, typ ast.Type, origin types.Origin) (Expr, error) {
	switch sig.Kind {
	case types.HTTPCallMethod, types.HTTPCallPath, types.HTTPCallQuery, types.HTTPCallBody:
		request, err := b.buildExpr(expr.Args[0])
		if err != nil {
			return nil, err
		}
		return &HTTPRequestStringExpr{Kind: sig.Kind, Request: request, Typ: typ, Origin: origin}, nil
	case types.HTTPCallHeader, types.HTTPCallParam, types.HTTPCallContext:
		request, err := b.buildExpr(expr.Args[0])
		if err != nil {
			return nil, err
		}
		name, err := b.buildExpr(expr.Args[1])
		if err != nil {
			return nil, err
		}
		return &HTTPRequestStringExpr{Kind: sig.Kind, Request: request, Name: name, Typ: typ, Origin: origin}, nil
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
