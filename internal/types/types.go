package types

import (
	"fmt"
	"strings"

	"github.com/claudioscheer/trux/internal/ast"
	"github.com/claudioscheer/trux/internal/stdlib"
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

type StructSig struct {
	Name        string
	DisplayName string
	Fields      []ast.StructField
	FieldMap    map[string]ast.Type
}

type Info struct {
	Funcs         map[string]FuncSig
	Kernels       map[string]FuncSig
	Structs       map[string]StructSig
	Locals        map[*ast.FuncDecl]map[string]ast.Type
	ExprTypes     map[ast.Expression]ast.Type
	ExprOrigins   map[ast.Expression]Origin
	ResolvedCalls map[*ast.CallExpr]FuncSig
	PrintCalls    map[*ast.CallExpr][]ast.Type
	AssertCalls   map[*ast.CallExpr]struct{}
	LenCalls      map[*ast.CallExpr]ast.Type
	AppendCalls   map[*ast.CallExpr]AppendSig
	CloneCalls    map[*ast.CallExpr]CloneSig
	DeleteCalls   map[*ast.CallExpr]*ast.MapType
	KeysCalls     map[*ast.CallExpr]*ast.MapType
	ValuesCalls   map[*ast.CallExpr]*ast.MapType
	MapLookups    map[*ast.IndexExpr]*ast.MapType
	IOCalls       map[*ast.CallExpr]IOCallSig
	HTTPCalls     map[*ast.CallExpr]HTTPCallSig
	GPUCalls      map[*ast.CallExpr]GPUCallSig
	SQLiteCalls   map[*ast.CallExpr]SQLiteCallSig
	JSONCalls     map[*ast.CallExpr]JSONCallSig
	JWTCalls      map[*ast.CallExpr]JWTCallSig
	KernelAxes    map[string]int
	UsesSQLite    bool
	UsesJSON      bool
	UsesJWT       bool
	UsesEnv       bool
}

type AppendSig struct {
	ListType ast.Type
	ElemType ast.Type
}

type CloneSig struct {
	Type ast.Type
}

type IOCallKind string

const (
	IOCallReadLine           IOCallKind = IOCallKind(stdlib.CallReadLine)
	IOCallReadInt            IOCallKind = IOCallKind(stdlib.CallReadInt)
	IOCallReadFloat          IOCallKind = IOCallKind(stdlib.CallReadFloat)
	IOCallReadBool           IOCallKind = IOCallKind(stdlib.CallReadBool)
	IOCallReadFile           IOCallKind = IOCallKind(stdlib.CallReadFile)
	IOCallWriteFile          IOCallKind = IOCallKind(stdlib.CallWriteFile)
	IOCallCreateDir          IOCallKind = IOCallKind(stdlib.CallCreateDir)
	IOCallRemove             IOCallKind = IOCallKind(stdlib.CallRemove)
	IOCallReadDir            IOCallKind = IOCallKind(stdlib.CallReadDir)
	IOCallExists             IOCallKind = IOCallKind(stdlib.CallExists)
	IOCallIsFile             IOCallKind = IOCallKind(stdlib.CallIsFile)
	IOCallIsDir              IOCallKind = IOCallKind(stdlib.CallIsDir)
	IOCallEnv                IOCallKind = IOCallKind(stdlib.CallEnv)
	IOCallReadCSV            IOCallKind = IOCallKind(stdlib.CallReadCSV)
	IOCallWriteCSV           IOCallKind = IOCallKind(stdlib.CallWriteCSV)
	IOCallImageWidth         IOCallKind = IOCallKind(stdlib.CallImageWidth)
	IOCallImageHeight        IOCallKind = IOCallKind(stdlib.CallImageHeight)
	IOCallReadPPM            IOCallKind = IOCallKind(stdlib.CallReadPPM)
	IOCallWritePPM           IOCallKind = IOCallKind(stdlib.CallWritePPM)
	IOCallTimeNowUnixMillis  IOCallKind = IOCallKind(stdlib.CallTimeNowUnixMillis)
	IOCallTimeMonotonicNanos IOCallKind = IOCallKind(stdlib.CallTimeMonotonicNanos)
	IOCallTimeSleepMillis    IOCallKind = IOCallKind(stdlib.CallTimeSleepMillis)
	IOCallErrorsNew          IOCallKind = IOCallKind(stdlib.CallErrorsNew)
	IOCallErrorsMessage      IOCallKind = IOCallKind(stdlib.CallErrorsMessage)
)

type IOCallSig struct {
	Kind IOCallKind
}

type SQLiteCallKind string

const (
	SQLiteCallOpen            SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteOpen)
	SQLiteCallOpenThreadLocal SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteOpenThreadLocal)
	SQLiteCallClose           SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteClose)
	SQLiteCallExec            SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteExec)
	SQLiteCallQuery           SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteQuery)
	SQLiteCallLastInsertRowID SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteLastInsertRowID)
	SQLiteCallChanges         SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteChanges)
	SQLiteCallTotalChanges    SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteTotalChanges)
	SQLiteCallBegin           SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteBegin)
	SQLiteCallCommit          SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteCommit)
	SQLiteCallRollback        SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteRollback)
	SQLiteCallPrepare         SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLitePrepare)
	SQLiteCallStep            SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteStep)
	SQLiteCallReset           SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteReset)
	SQLiteCallClearBindings   SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteClearBindings)
	SQLiteCallFinalize        SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteFinalize)
	SQLiteCallBindInt         SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteBindInt)
	SQLiteCallBindFloat       SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteBindFloat)
	SQLiteCallBindText        SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteBindText)
	SQLiteCallBindNull        SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteBindNull)
	SQLiteCallBindIndex       SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteBindIndex)
	SQLiteCallColumnCount     SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteColumnCount)
	SQLiteCallColumnName      SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteColumnName)
	SQLiteCallColumnIndex     SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteColumnIndex)
	SQLiteCallColumnType      SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteColumnType)
	SQLiteCallColumnInt       SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteColumnInt)
	SQLiteCallColumnFloat     SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteColumnFloat)
	SQLiteCallColumnText      SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteColumnText)
	SQLiteCallColumnIsNull    SQLiteCallKind = SQLiteCallKind(stdlib.CallSQLiteColumnIsNull)
)

type SQLiteCallSig struct {
	Kind SQLiteCallKind
}

type JSONCallKind string

const (
	JSONCallParse       JSONCallKind = JSONCallKind(stdlib.CallJSONParse)
	JSONCallRoot        JSONCallKind = JSONCallKind(stdlib.CallJSONRoot)
	JSONCallGet         JSONCallKind = JSONCallKind(stdlib.CallJSONGet)
	JSONCallAt          JSONCallKind = JSONCallKind(stdlib.CallJSONAt)
	JSONCallLen         JSONCallKind = JSONCallKind(stdlib.CallJSONLen)
	JSONCallIsObject    JSONCallKind = JSONCallKind(stdlib.CallJSONIsObject)
	JSONCallIsArray     JSONCallKind = JSONCallKind(stdlib.CallJSONIsArray)
	JSONCallIsString    JSONCallKind = JSONCallKind(stdlib.CallJSONIsString)
	JSONCallIsInt       JSONCallKind = JSONCallKind(stdlib.CallJSONIsInt)
	JSONCallIsFloat     JSONCallKind = JSONCallKind(stdlib.CallJSONIsFloat)
	JSONCallIsBool      JSONCallKind = JSONCallKind(stdlib.CallJSONIsBool)
	JSONCallIsNull      JSONCallKind = JSONCallKind(stdlib.CallJSONIsNull)
	JSONCallGetString   JSONCallKind = JSONCallKind(stdlib.CallJSONGetString)
	JSONCallGetInt      JSONCallKind = JSONCallKind(stdlib.CallJSONGetInt)
	JSONCallGetFloat    JSONCallKind = JSONCallKind(stdlib.CallJSONGetFloat)
	JSONCallGetBool     JSONCallKind = JSONCallKind(stdlib.CallJSONGetBool)
	JSONCallFieldString JSONCallKind = JSONCallKind(stdlib.CallJSONFieldString)
	JSONCallFieldInt    JSONCallKind = JSONCallKind(stdlib.CallJSONFieldInt)
	JSONCallFieldFloat  JSONCallKind = JSONCallKind(stdlib.CallJSONFieldFloat)
	JSONCallFieldBool   JSONCallKind = JSONCallKind(stdlib.CallJSONFieldBool)
	JSONCallDecodeMap   JSONCallKind = JSONCallKind(stdlib.CallJSONDecodeMap)
	JSONCallEncodeMap   JSONCallKind = JSONCallKind(stdlib.CallJSONEncodeMap)
	JSONCallStringify   JSONCallKind = JSONCallKind(stdlib.CallJSONStringify)
	JSONCallValid       JSONCallKind = JSONCallKind(stdlib.CallJSONValid)
	JSONCallNewDoc      JSONCallKind = JSONCallKind(stdlib.CallJSONNewDoc)
	JSONCallNewObject   JSONCallKind = JSONCallKind(stdlib.CallJSONNewObject)
	JSONCallNewArray    JSONCallKind = JSONCallKind(stdlib.CallJSONNewArray)
	JSONCallSetRoot     JSONCallKind = JSONCallKind(stdlib.CallJSONSetRoot)
	JSONCallAddString   JSONCallKind = JSONCallKind(stdlib.CallJSONAddString)
	JSONCallAddInt      JSONCallKind = JSONCallKind(stdlib.CallJSONAddInt)
	JSONCallAddFloat    JSONCallKind = JSONCallKind(stdlib.CallJSONAddFloat)
	JSONCallAddBool     JSONCallKind = JSONCallKind(stdlib.CallJSONAddBool)
	JSONCallAddNull     JSONCallKind = JSONCallKind(stdlib.CallJSONAddNull)
	JSONCallAddVal      JSONCallKind = JSONCallKind(stdlib.CallJSONAddVal)
	JSONCallAppendVal   JSONCallKind = JSONCallKind(stdlib.CallJSONAppendVal)
	JSONCallWrite       JSONCallKind = JSONCallKind(stdlib.CallJSONWrite)
)

type JSONCallSig struct {
	Kind JSONCallKind
}

type JWTCallKind string

const (
	JWTCallSign      JWTCallKind = JWTCallKind(stdlib.CallJWTSign)
	JWTCallVerify    JWTCallKind = JWTCallKind(stdlib.CallJWTVerify)
	JWTCallGetString JWTCallKind = JWTCallKind(stdlib.CallJWTGetString)
	JWTCallGetInt    JWTCallKind = JWTCallKind(stdlib.CallJWTGetInt)
	JWTCallGetBool   JWTCallKind = JWTCallKind(stdlib.CallJWTGetBool)
	JWTCallGetFloat  JWTCallKind = JWTCallKind(stdlib.CallJWTGetFloat)
)

type JWTCallSig struct {
	Kind JWTCallKind
}

type HTTPCallKind string

const (
	HTTPCallListen    HTTPCallKind = HTTPCallKind(stdlib.CallHTTPListen)
	HTTPCallServe     HTTPCallKind = HTTPCallKind(stdlib.CallHTTPServe)
	HTTPCallUse       HTTPCallKind = HTTPCallKind(stdlib.CallHTTPUse)
	HTTPCallUseAt     HTTPCallKind = HTTPCallKind(stdlib.CallHTTPUseAt)
	HTTPCallGet       HTTPCallKind = HTTPCallKind(stdlib.CallHTTPGet)
	HTTPCallPost      HTTPCallKind = HTTPCallKind(stdlib.CallHTTPPost)
	HTTPCallPut       HTTPCallKind = HTTPCallKind(stdlib.CallHTTPPut)
	HTTPCallDelete    HTTPCallKind = HTTPCallKind(stdlib.CallHTTPDelete)
	HTTPCallPatch     HTTPCallKind = HTTPCallKind(stdlib.CallHTTPPatch)
	HTTPCallRoute     HTTPCallKind = HTTPCallKind(stdlib.CallHTTPRoute)
	HTTPCallNotFound  HTTPCallKind = HTTPCallKind(stdlib.CallHTTPNotFound)
	HTTPCallParam     HTTPCallKind = HTTPCallKind(stdlib.CallHTTPParam)
	HTTPCallSet       HTTPCallKind = HTTPCallKind(stdlib.CallHTTPSet)
	HTTPCallContext   HTTPCallKind = HTTPCallKind(stdlib.CallHTTPContext)
	HTTPCallSetHeader HTTPCallKind = HTTPCallKind(stdlib.CallHTTPSetHeader)
	HTTPCallText      HTTPCallKind = HTTPCallKind(stdlib.CallHTTPText)
	HTTPCallJSON      HTTPCallKind = HTTPCallKind(stdlib.CallHTTPJSON)
	HTTPCallStatus    HTTPCallKind = HTTPCallKind(stdlib.CallHTTPStatus)
	HTTPCallMethod    HTTPCallKind = HTTPCallKind(stdlib.CallHTTPMethod)
	HTTPCallPath      HTTPCallKind = HTTPCallKind(stdlib.CallHTTPPath)
	HTTPCallQuery     HTTPCallKind = HTTPCallKind(stdlib.CallHTTPQuery)
	HTTPCallHeader    HTTPCallKind = HTTPCallKind(stdlib.CallHTTPHeader)
	HTTPCallBody      HTTPCallKind = HTTPCallKind(stdlib.CallHTTPBody)
	HTTPCallRespond   HTTPCallKind = HTTPCallKind(stdlib.CallHTTPRespond)
)

type HTTPCallSig struct {
	Kind        HTTPCallKind
	HandlerName string
	HandlerSig  FuncSig
}

type GPUCallKind string

const (
	GPUCallAlloc        GPUCallKind = "alloc"
	GPUCallCopyToDevice GPUCallKind = "copyToDevice"
	GPUCallCopyToHost   GPUCallKind = "copyToHost"
	GPUCallFree         GPUCallKind = "free"
	GPUCallSync         GPUCallKind = "sync"
	GPUCallLaunch       GPUCallKind = "launch"
	GPUCallLaunch1D     GPUCallKind = "launch1d"
	GPUCallLaunch2D     GPUCallKind = "launch2d"
	GPUCallFrom         GPUCallKind = "from"
	GPUCallGlobalX      GPUCallKind = "globalX"
	GPUCallGlobalY      GPUCallKind = "globalY"
	GPUCallGlobalZ      GPUCallKind = "globalZ"
	GPUCallThreadX      GPUCallKind = "threadX"
	GPUCallThreadY      GPUCallKind = "threadY"
	GPUCallThreadZ      GPUCallKind = "threadZ"
	GPUCallBlockX       GPUCallKind = "blockX"
	GPUCallBlockY       GPUCallKind = "blockY"
	GPUCallBlockZ       GPUCallKind = "blockZ"
	GPUCallBlockDimX    GPUCallKind = "blockDimX"
	GPUCallBlockDimY    GPUCallKind = "blockDimY"
	GPUCallBlockDimZ    GPUCallKind = "blockDimZ"
)

type GPUCallSig struct {
	Kind       GPUCallKind
	BufferType ast.Type
	KernelName string
	KernelSig  FuncSig
}

type Origin string

const (
	OriginBorrowed   Origin = "borrowed"
	OriginScratch    Origin = "scratch"
	OriginOwned      Origin = "owned"
	OriginFrameOwned Origin = "frame-owned"
	OriginUnknown    Origin = "unknown"
)

type loopExit struct {
	scope     *scope
	breaks    []originSnap
	continues []originSnap
}

type checker struct {
	info       *Info
	loopDepth  int
	loops      []*loopExit
	deadCode   int
	reachesEnd bool
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

type originSnap struct {
	levels []map[string]Origin
}

func snapshotOrigins(s *scope) originSnap {
	var snap originSnap
	for scope := s; scope != nil; scope = scope.parent {
		level := make(map[string]Origin, len(scope.locals))
		for name, local := range scope.locals {
			level[name] = local.origin
		}
		snap.levels = append(snap.levels, level)
	}
	return snap
}

func restoreOrigins(s *scope, snap originSnap) {
	i := 0
	for scope := s; scope != nil; scope = scope.parent {
		level := snap.levels[i]
		for name := range scope.locals {
			origin, ok := level[name]
			if !ok {
				delete(scope.locals, name)
				continue
			}
			local := scope.locals[name]
			local.origin = origin
			scope.locals[name] = local
		}
		i++
	}
}

func joinExitOrigins(scope *scope, exits []originSnap) {
	if len(exits) == 0 {
		return
	}
	restoreOrigins(scope, exits[0])
	current := exits[0]
	for _, next := range exits[1:] {
		joinOrigins(scope, current, next)
		current = snapshotOrigins(scope)
	}
}

func joinOrigins(s *scope, left originSnap, right originSnap) {
	i := 0
	for scope := s; scope != nil; scope = scope.parent {
		for name, local := range scope.locals {
			local.origin = joinOrigin(left.levels[i][name], right.levels[i][name])
			scope.locals[name] = local
		}
		i++
	}
}

func joinOrigin(left Origin, right Origin) Origin {
	if left == right || right == "" {
		return left
	}
	if left == "" {
		return right
	}
	if originCopiesOut(left) || originCopiesOut(right) {
		if left == OriginFrameOwned || right == OriginFrameOwned {
			return OriginFrameOwned
		}
		return OriginScratch
	}
	if left == OriginBorrowed || right == OriginBorrowed {
		return OriginBorrowed
	}
	return left
}

func originCopiesOut(origin Origin) bool {
	switch origin {
	case OriginScratch, OriginFrameOwned, OriginUnknown:
		return true
	default:
		return false
	}
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
			Kernels:       map[string]FuncSig{},
			Structs:       map[string]StructSig{},
			Locals:        map[*ast.FuncDecl]map[string]ast.Type{},
			ExprTypes:     map[ast.Expression]ast.Type{},
			ExprOrigins:   map[ast.Expression]Origin{},
			ResolvedCalls: map[*ast.CallExpr]FuncSig{},
			PrintCalls:    map[*ast.CallExpr][]ast.Type{},
			AssertCalls:   map[*ast.CallExpr]struct{}{},
			LenCalls:      map[*ast.CallExpr]ast.Type{},
			AppendCalls:   map[*ast.CallExpr]AppendSig{},
			CloneCalls:    map[*ast.CallExpr]CloneSig{},
			DeleteCalls:   map[*ast.CallExpr]*ast.MapType{},
			KeysCalls:     map[*ast.CallExpr]*ast.MapType{},
			ValuesCalls:   map[*ast.CallExpr]*ast.MapType{},
			MapLookups:    map[*ast.IndexExpr]*ast.MapType{},
			IOCalls:       map[*ast.CallExpr]IOCallSig{},
			HTTPCalls:     map[*ast.CallExpr]HTTPCallSig{},
			GPUCalls:      map[*ast.CallExpr]GPUCallSig{},
			SQLiteCalls:   map[*ast.CallExpr]SQLiteCallSig{},
			JSONCalls:     map[*ast.CallExpr]JSONCallSig{},
			JWTCalls:      map[*ast.CallExpr]JWTCallSig{},
			KernelAxes:    map[string]int{},
		},
	}

	for _, imp := range program.Imports {
		if imp.Path == "sqlite" {
			c.info.UsesSQLite = true
			c.registerSQLiteStructs()
		}
		if imp.Path == "json" {
			c.info.UsesJSON = true
			c.registerJSONStructs()
		}
		if imp.Path == "jwt" {
			c.info.UsesJWT = true
			c.registerJWTStructs()
		}
	}

	if err := c.collectStructs(program); err != nil {
		return nil, err
	}
	if err := c.collectFunctions(program); err != nil {
		return nil, err
	}
	if err := c.checkMain(program); err != nil {
		return nil, err
	}
	for _, fn := range program.Functions {
		var err error
		if fn.Kernel {
			err = c.checkKernelFunc(fn)
		} else {
			err = c.checkFunc(fn)
		}
		if err != nil {
			return nil, err
		}
	}

	return c.info, nil
}

func (c *checker) registerSQLiteStructs() {
	if _, ok := c.info.Structs["trux_sqlite_db"]; !ok {
		c.info.Structs["trux_sqlite_db"] = StructSig{
			Name:        "trux_sqlite_db",
			DisplayName: "sqlite.DB",
			Fields:      nil,
			FieldMap:    map[string]ast.Type{},
		}
	}
	if _, ok := c.info.Structs["trux_sqlite_stmt"]; !ok {
		c.info.Structs["trux_sqlite_stmt"] = StructSig{
			Name:        "trux_sqlite_stmt",
			DisplayName: "sqlite.Stmt",
			Fields:      nil,
			FieldMap:    map[string]ast.Type{},
		}
	}
}

func (c *checker) registerJSONStructs() {
	if _, ok := c.info.Structs["trux_json_doc"]; !ok {
		c.info.Structs["trux_json_doc"] = StructSig{
			Name:        "trux_json_doc",
			DisplayName: "json.Doc",
			Fields:      nil,
			FieldMap:    map[string]ast.Type{},
		}
	}
	if _, ok := c.info.Structs["trux_json_val"]; !ok {
		c.info.Structs["trux_json_val"] = StructSig{
			Name:        "trux_json_val",
			DisplayName: "json.Val",
			Fields:      nil,
			FieldMap:    map[string]ast.Type{},
		}
	}
	if _, ok := c.info.Structs["trux_json_mut_doc"]; !ok {
		c.info.Structs["trux_json_mut_doc"] = StructSig{
			Name:        "trux_json_mut_doc",
			DisplayName: "json.MutDoc",
			Fields:      nil,
			FieldMap:    map[string]ast.Type{},
		}
	}
	if _, ok := c.info.Structs["trux_json_mut_val"]; !ok {
		c.info.Structs["trux_json_mut_val"] = StructSig{
			Name:        "trux_json_mut_val",
			DisplayName: "json.MutVal",
			Fields:      nil,
			FieldMap:    map[string]ast.Type{},
		}
	}
}

func (c *checker) registerJWTStructs() {
	if _, ok := c.info.Structs["trux_jwt_claims"]; !ok {
		c.info.Structs["trux_jwt_claims"] = StructSig{
			Name:        "trux_jwt_claims",
			DisplayName: "jwt.Claims",
			Fields:      nil,
			FieldMap:    map[string]ast.Type{},
		}
	}
}

func (c *checker) collectStructs(program *ast.Program) error {
	for _, st := range program.Structs {
		targetName := st.TargetName()
		if _, exists := c.info.Structs[targetName]; exists {
			return typeError(st.Pos, "duplicate struct %q", st.Name)
		}
		fieldMap := make(map[string]ast.Type, len(st.Fields))
		for _, field := range st.Fields {
			if _, exists := fieldMap[field.Name]; exists {
				return typeError(field.Pos, "duplicate field %q in struct %q", field.Name, st.Name)
			}
			fieldMap[field.Name] = field.Type
		}
		c.info.Structs[targetName] = StructSig{
			Name:        targetName,
			DisplayName: st.Name,
			Fields:      st.Fields,
			FieldMap:    fieldMap,
		}
	}

	for _, st := range program.Structs {
		targetName := st.TargetName()
		sig := c.info.Structs[targetName]
		for i, field := range st.Fields {
			if err := c.validateType(field.Pos, field.Type); err != nil {
				return err
			}
			sig.Fields[i].Type = field.Type
			sig.FieldMap[field.Name] = field.Type
		}
		c.info.Structs[targetName] = sig
	}

	for _, st := range program.Structs {
		visited := map[string]bool{}
		if c.hasValueStructCycle(st.TargetName(), st.TargetName(), visited) {
			return typeError(st.Pos, "invalid recursive struct %q", st.Name)
		}
	}

	return nil
}

func (c *checker) hasValueStructCycle(start string, current string, visited map[string]bool) bool {
	sig, ok := c.info.Structs[current]
	if !ok {
		return false
	}
	for _, field := range sig.Fields {
		for _, embedded := range valueEmbeddedStructs(field.Type) {
			if embedded == start {
				return true
			}
			if !visited[embedded] {
				visited[embedded] = true
				if c.hasValueStructCycle(start, embedded, visited) {
					return true
				}
			}
		}
	}
	return false
}

func valueEmbeddedStructs(typ ast.Type) []string {
	switch typ := typ.(type) {
	case *ast.StructType:
		return []string{typ.TargetName()}
	case *ast.ArrayType:
		return valueEmbeddedStructs(typ.Elem)
	default:
		return nil
	}
}

func (c *checker) collectFunctions(program *ast.Program) error {
	for _, fn := range program.Functions {
		target := c.info.Funcs
		if fn.Kernel {
			target = c.info.Kernels
		}
		if _, exists := target[fn.Name]; exists {
			return typeError(fn.Pos, "duplicate function %q", fn.Name)
		}
		if !fn.Kernel {
			if fn.ReturnType == nil {
				return typeError(fn.Pos, "function %q must declare return type", fn.Name)
			}
			if err := c.validateReturnType(fn.Pos, fn.ReturnType); err != nil {
				return err
			}
		}
		if _, exists := c.info.Funcs[fn.Name]; exists {
			return typeError(fn.Pos, "duplicate function %q", fn.Name)
		}
		if _, exists := c.info.Kernels[fn.Name]; exists {
			return typeError(fn.Pos, "duplicate function %q", fn.Name)
		}
		for _, param := range fn.Params {
			if err := c.validateType(param.Pos, param.Type); err != nil {
				return err
			}
			if err := validateMutableParam(fn, param); err != nil {
				return err
			}
		}

		target[fn.Name] = FuncSig{
			Name:       fn.Name,
			Params:     fn.Params,
			ReturnType: fn.ReturnType,
		}
	}

	return nil
}

func (c *checker) checkKernelFunc(fn *ast.FuncDecl) error {
	c.info.Locals[fn] = map[string]ast.Type{}
	locals := newScope(nil)
	for _, param := range fn.Params {
		if _, exists := c.info.Locals[fn][param.Name]; exists {
			return typeError(fn.Pos, "duplicate parameter %q", param.Name)
		}
		if _, ok := param.Type.(*ast.GPUBufferType); !ok && !ast.TypeEqual(param.Type, ast.IntType) && !ast.TypeEqual(param.Type, ast.FloatType) && !ast.TypeEqual(param.Type, ast.BoolType) {
			return typeError(param.Pos, "kernel parameter %q has unsupported type %s", param.Name, param.Type)
		}
		c.info.Locals[fn][param.Name] = param.Type
		locals.locals[param.Name] = localInfo{typ: param.Type, origin: OriginOwned}
	}
	for _, stmt := range fn.Body.Statements {
		if err := c.checkKernelStmt(fn, locals, stmt); err != nil {
			return err
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
			origin: originForParam(param),
		}
	}

	for _, stmt := range fn.Body.Statements {
		if err := c.checkStmt(fn, locals, stmt); err != nil {
			return err
		}
	}
	if !blockTerminates(fn.Body) {
		return typeError(fn.Pos, "function %q must return on all paths", fn.Name)
	}

	return nil
}

func blockTerminates(block ast.Block) bool {
	for _, stmt := range block.Statements {
		if stmtTerminates(stmt) {
			return true
		}
	}

	return false
}

func stmtTerminates(stmt ast.Statement) bool {
	switch stmt := stmt.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.IfStmt:
		return stmt.Else != nil && blockTerminates(stmt.Then) && blockTerminates(*stmt.Else)
	default:
		return false
	}
}

func (c *checker) checkStmtList(fn *ast.FuncDecl, locals *scope, stmts []ast.Statement) (bool, error) {
	reachesEnd := true
	for _, stmt := range stmts {
		if !reachesEnd {
			saved := snapshotOrigins(locals)
			c.deadCode++
			err := c.checkStmt(fn, locals, stmt)
			c.deadCode--
			restoreOrigins(locals, saved)
			if err != nil {
				return false, err
			}
			continue
		}
		if err := c.checkStmt(fn, locals, stmt); err != nil {
			return false, err
		}
		if !c.reachesEnd {
			reachesEnd = false
		}
	}
	return reachesEnd, nil
}

func (c *checker) noteLoopJump(continues bool) {
	if c.deadCode > 0 || len(c.loops) == 0 {
		return
	}
	frame := c.loops[len(c.loops)-1]
	snap := snapshotOrigins(frame.scope)
	if continues {
		frame.continues = append(frame.continues, snap)
		return
	}
	frame.breaks = append(frame.breaks, snap)
}

func (c *checker) checkMultiValueExpr(locals *scope, expr ast.Expression) (ast.Type, error) {
	if idx, ok := expr.(*ast.IndexExpr); ok {
		collectionType, err := c.checkExpr(locals, idx.Collection, false)
		if err != nil {
			return nil, err
		}
		if mapType, isMap := collectionType.(*ast.MapType); isMap {
			keyType, err := c.checkExpr(locals, idx.Index, false)
			if err != nil {
				return nil, err
			}
			keyType = c.coerceNil(idx.Index, keyType, mapType.Key)
			if !ast.TypeEqual(keyType, mapType.Key) {
				return nil, typeError(idx.Index.Pos(), "map key has type %s, want %s", keyType, mapType.Key)
			}
			tupleType := &ast.TupleType{Types: []ast.Type{mapType.Value, ast.BoolType}}
			c.info.MapLookups[idx] = mapType
			c.setExpr(idx, tupleType, indexOrigin(mapType, c.exprOrigin(idx.Collection)))
			return tupleType, nil
		}
	}
	return c.checkExpr(locals, expr, false)
}

func (c *checker) coerceNil(expr ast.Expression, got ast.Type, want ast.Type) ast.Type {
	if _, isNil := expr.(*ast.NilLiteral); isNil {
		if _, isPtr := want.(*ast.PointerType); isPtr {
			c.setExpr(expr, want, OriginOwned)
			return want
		}
	}
	return got
}

func (c *checker) isAddressable(expr ast.Expression) bool {
	switch expr := expr.(type) {
	case *ast.IdentExpr:
		return true
	case *ast.IndexExpr:
		if _, isMap := c.info.ExprTypes[expr.Collection].(*ast.MapType); isMap {
			return false
		}
		return !ast.TypeEqual(c.info.ExprTypes[expr.Collection], ast.StringType)
	case *ast.FieldExpr:
		if _, isPtr := c.info.ExprTypes[expr.Object].(*ast.PointerType); isPtr {
			return true
		}
		return c.isAddressable(expr.Object)
	case *ast.UnaryExpr:
		return expr.Operator == "*"
	default:
		return false
	}
}

func (c *checker) checkStmt(fn *ast.FuncDecl, locals *scope, stmt ast.Statement) error {
	c.reachesEnd = true
	switch stmt := stmt.(type) {
	case *ast.LetStmt:
		if len(stmt.Bindings) > 1 {
			valType, err := c.checkMultiValueExpr(locals, stmt.Value)
			if err != nil {
				return err
			}
			tuple, ok := valType.(*ast.TupleType)
			if !ok {
				return typeError(stmt.Value.Pos(), "cannot unpack non-tuple type %s into %d variables", valType, len(stmt.Bindings))
			}
			if len(stmt.Bindings) != len(tuple.Types) {
				return typeError(stmt.Start, "cannot unpack %d values into %d variables", len(tuple.Types), len(stmt.Bindings))
			}
			for i, binding := range stmt.Bindings {
				elemType := tuple.Types[i]
				if binding.Name == "_" {
					if binding.Type != nil {
						if err := c.validateType(binding.Pos, binding.Type); err != nil {
							return err
						}
						if !ast.TypeEqual(binding.Type, elemType) {
							return typeError(binding.Pos, "cannot discard %s with type %s", elemType, binding.Type)
						}
					}
					continue
				}
				if _, exists := locals.locals[binding.Name]; exists {
					return typeError(binding.Pos, "duplicate local variable %q", binding.Name)
				}
				if binding.Type != nil {
					if err := c.validateType(binding.Pos, binding.Type); err != nil {
						return err
					}
					if !ast.TypeEqual(binding.Type, elemType) {
						return typeError(binding.Pos, "cannot assign %s to %s", elemType, binding.Type)
					}
				}
				if _, exists := c.info.Locals[fn][binding.Name]; !exists {
					c.info.Locals[fn][binding.Name] = elemType
				}
				locals.locals[binding.Name] = localInfo{
					typ:    elemType,
					origin: originForUnpackedValue(elemType),
				}
			}
			return nil
		}
		if stmt.Name == "_" {
			valType, err := c.checkExpr(locals, stmt.Value, false)
			if err != nil {
				return err
			}
			if stmt.Type != nil {
				if err := c.validateType(stmt.Start, stmt.Type); err != nil {
					return err
				}
				valType = c.coerceNil(stmt.Value, valType, stmt.Type)
				if !ast.TypeEqual(valType, stmt.Type) {
					return typeError(stmt.Start, "cannot assign %s to _ of type %s", valType, stmt.Type)
				}
			}
			return nil
		}
		if _, exists := locals.locals[stmt.Name]; exists {
			return typeError(stmt.Start, "duplicate local variable %q", stmt.Name)
		}
		if err := c.validateType(stmt.Start, stmt.Type); err != nil {
			return err
		}

		valueType, err := c.checkLetValue(locals, stmt.Type, stmt.Value)
		if err != nil {
			return err
		}
		valueType = c.coerceNil(stmt.Value, valueType, stmt.Type)
		if !ast.TypeEqual(valueType, stmt.Type) {
			return typeError(stmt.Value.Pos(), "cannot assign %s to %s", valueType, stmt.Type)
		}

		if _, exists := c.info.Locals[fn][stmt.Name]; !exists {
			c.info.Locals[fn][stmt.Name] = stmt.Type
		}
		locals.locals[stmt.Name] = localInfo{
			typ:    stmt.Type,
			origin: c.exprOrigin(stmt.Value),
		}
		return nil
	case *ast.AssignStmt:
		if len(stmt.Targets) > 1 {
			valueType, err := c.checkMultiValueExpr(locals, stmt.Value)
			if err != nil {
				return err
			}
			tuple, ok := valueType.(*ast.TupleType)
			if !ok {
				return typeError(stmt.Value.Pos(), "cannot unpack non-tuple type %s into %d variables", valueType, len(stmt.Targets))
			}
			if len(stmt.Targets) != len(tuple.Types) {
				return typeError(stmt.Start, "cannot unpack %d values into %d variables", len(tuple.Types), len(stmt.Targets))
			}
			for i, target := range stmt.Targets {
				elemType := tuple.Types[i]
				if target.Name == "_" {
					continue
				}
				local, ok := locals.lookup(target.Name)
				if !ok {
					return typeError(target.Pos, "undefined variable %q", target.Name)
				}
				if !ast.TypeEqual(elemType, local.typ) {
					return typeError(target.Pos, "cannot assign %s to %s", elemType, local.typ)
				}
				local.origin = originForUnpackedValue(elemType)
				locals.assign(target.Name, local)
			}
			return nil
		}
		if stmt.Name == "_" {
			_, err := c.checkExpr(locals, stmt.Value, false)
			return err
		}
		local, ok := locals.lookup(stmt.Name)
		if !ok {
			return typeError(stmt.Start, "undefined variable %q", stmt.Name)
		}
		valueType, err := c.checkExpr(locals, stmt.Value, false)
		if err != nil {
			return err
		}
		valueType = c.coerceNil(stmt.Value, valueType, local.typ)
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
		if mapType, isMap := collectionType.(*ast.MapType); isMap {
			indexType = c.coerceNil(stmt.Target.Index, indexType, mapType.Key)
			if !ast.TypeEqual(indexType, mapType.Key) {
				return typeError(stmt.Target.Index.Pos(), "map key has type %s, want %s", indexType, mapType.Key)
			}
			if err := c.checkCanMutate(stmt.Target.Collection); err != nil {
				return err
			}
			valueType, err := c.checkExpr(locals, stmt.Value, false)
			if err != nil {
				return err
			}
			valueType = c.coerceNil(stmt.Value, valueType, mapType.Value)
			if !ast.TypeEqual(valueType, mapType.Value) {
				return typeError(stmt.Value.Pos(), "cannot assign %s to %s map value", valueType, mapType.Value)
			}
			c.setExpr(stmt.Target, mapType.Value, indexOrigin(mapType, c.exprOrigin(stmt.Target.Collection)))
			return nil
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
		valueType = c.coerceNil(stmt.Value, valueType, elemType)
		if !ast.TypeEqual(valueType, elemType) {
			return typeError(stmt.Value.Pos(), "cannot assign %s to %s element", valueType, elemType)
		}
		c.setExpr(stmt.Target, elemType, indexOrigin(collectionType, c.exprOrigin(stmt.Target.Collection)))
		return nil
	case *ast.FieldAssignStmt:
		fieldType, err := c.checkExpr(locals, stmt.Target, false)
		if err != nil {
			return err
		}
		if _, isPtr := c.info.ExprTypes[stmt.Target.Object].(*ast.PointerType); !isPtr && !c.isAddressable(stmt.Target.Object) {
			return typeError(stmt.Target.Pos(), "cannot assign to field %q of non-addressable struct value", stmt.Target.Field)
		}
		valueType, err := c.checkExpr(locals, stmt.Value, false)
		if err != nil {
			return err
		}
		valueType = c.coerceNil(stmt.Value, valueType, fieldType)
		if !ast.TypeEqual(valueType, fieldType) {
			return typeError(stmt.Value.Pos(), "cannot assign %s to field %q of type %s", valueType, stmt.Target.Field, fieldType)
		}
		return nil
	case *ast.DerefAssignStmt:
		ptrType, err := c.checkExpr(locals, stmt.Target.Operand, false)
		if err != nil {
			return err
		}
		ptr, ok := ptrType.(*ast.PointerType)
		if !ok {
			return typeError(stmt.Target.Pos(), "cannot dereference non-pointer type %s", ptrType)
		}
		c.setExpr(stmt.Target, ptr.Elem, OriginOwned)
		valueType, err := c.checkExpr(locals, stmt.Value, false)
		if err != nil {
			return err
		}
		valueType = c.coerceNil(stmt.Value, valueType, ptr.Elem)
		if !ast.TypeEqual(valueType, ptr.Elem) {
			return typeError(stmt.Value.Pos(), "cannot assign %s to %s", valueType, ptr.Elem)
		}
		return nil
	case *ast.ReturnStmt:
		values := stmt.Values
		if len(values) == 0 && stmt.Value != nil {
			values = []ast.Expression{stmt.Value}
		}
		if fn.ReturnType == nil {
			if len(values) > 0 {
				return typeError(stmt.Start, "cannot return value from void function")
			}
			c.reachesEnd = false
			return nil
		}
		if tuple, isTuple := fn.ReturnType.(*ast.TupleType); isTuple {
			if len(values) == 1 {
				valType, err := c.checkExpr(locals, values[0], false)
				if err != nil {
					return err
				}
				if !ast.TypeEqual(valType, tuple) {
					return typeError(values[0].Pos(), "cannot return %s from function returning %s", valType, tuple)
				}
			} else {
				if len(values) != len(tuple.Types) {
					return typeError(stmt.Start, "function returns %d values, but return statement has %d", len(tuple.Types), len(values))
				}
				for i, valExpr := range values {
					valType, err := c.checkExpr(locals, valExpr, false)
					if err != nil {
						return err
					}
					valType = c.coerceNil(valExpr, valType, tuple.Types[i])
					if !ast.TypeEqual(valType, tuple.Types[i]) {
						return typeError(valExpr.Pos(), "cannot return %s from function returning %s", valType, tuple.Types[i])
					}
				}
			}
		} else {
			if len(values) != 1 {
				return typeError(stmt.Start, "function returns 1 value, but return statement has %d", len(values))
			}
			valueType, err := c.checkExpr(locals, values[0], false)
			if err != nil {
				return err
			}
			valueType = c.coerceNil(values[0], valueType, fn.ReturnType)
			if !ast.TypeEqual(valueType, fn.ReturnType) {
				return typeError(values[0].Pos(), "cannot return %s from function returning %s", valueType, fn.ReturnType)
			}
		}
		c.reachesEnd = false
		return nil
	case *ast.IfStmt:
		conditionType, err := c.checkExpr(locals, stmt.Condition, false)
		if err != nil {
			return err
		}
		if !ast.TypeEqual(conditionType, ast.BoolType) {
			return typeError(stmt.Condition.Pos(), "if condition must be bool, got %s", conditionType)
		}
		before := snapshotOrigins(locals)
		thenScope := newScope(locals)
		thenReaches, err := c.checkStmtList(fn, thenScope, stmt.Then.Statements)
		if err != nil {
			return err
		}
		afterThen := snapshotOrigins(locals)
		restoreOrigins(locals, before)
		exits := []originSnap{}
		if thenReaches {
			exits = append(exits, afterThen)
		}
		if stmt.Else != nil {
			elseScope := newScope(locals)
			elseReaches, err := c.checkStmtList(fn, elseScope, stmt.Else.Statements)
			if err != nil {
				return err
			}
			if elseReaches {
				exits = append(exits, snapshotOrigins(locals))
			}
			restoreOrigins(locals, before)
		} else {
			exits = append(exits, before)
		}
		if len(exits) == 0 {
			c.reachesEnd = false
			return nil
		}
		joinExitOrigins(locals, exits)
		c.reachesEnd = true
		return nil
	case *ast.ForStmt:
		loopScope := newScope(locals)
		if stmt.Init != nil {
			if err := c.checkStmt(fn, loopScope, stmt.Init); err != nil {
				return err
			}
		}
		if stmt.Condition != nil {
			conditionType, err := c.checkExpr(loopScope, stmt.Condition, false)
			if err != nil {
				return err
			}
			if !ast.TypeEqual(conditionType, ast.BoolType) {
				return typeError(stmt.Condition.Pos(), "for condition must be bool, got %s", conditionType)
			}
		}
		before := snapshotOrigins(loopScope)
		frame := &loopExit{scope: loopScope}
		c.loops = append(c.loops, frame)
		c.loopDepth++
		bodyScope := newScope(loopScope)
		bodyReaches, err := c.checkStmtList(fn, bodyScope, stmt.Body.Statements)
		if err != nil {
			c.loopDepth--
			c.loops = c.loops[:len(c.loops)-1]
			return err
		}
		// Break leaves before the post. Continue and a normal body end run the post.
		incoming := []originSnap{}
		if bodyReaches {
			incoming = append(incoming, snapshotOrigins(loopScope))
		}
		incoming = append(incoming, frame.continues...)
		exits := []originSnap{before}
		exits = append(exits, frame.breaks...)
		if stmt.Post == nil {
			exits = append(exits, incoming...)
		} else {
			for _, in := range incoming {
				restoreOrigins(loopScope, in)
				if err := c.checkStmt(fn, loopScope, stmt.Post); err != nil {
					c.loopDepth--
					c.loops = c.loops[:len(c.loops)-1]
					return err
				}
				exits = append(exits, snapshotOrigins(loopScope))
			}
		}
		c.loopDepth--
		c.loops = c.loops[:len(c.loops)-1]
		joinExitOrigins(loopScope, exits)
		c.reachesEnd = true
		return nil
	case *ast.BreakStmt:
		if c.loopDepth == 0 {
			return typeError(stmt.Start, "break outside of a loop")
		}
		c.noteLoopJump(false)
		c.reachesEnd = false
		return nil
	case *ast.ContinueStmt:
		if c.loopDepth == 0 {
			return typeError(stmt.Start, "continue outside of a loop")
		}
		c.noteLoopJump(true)
		c.reachesEnd = false
		return nil
	case *ast.ExprStmt:
		typ, err := c.checkExpr(locals, stmt.Expr, true)
		if err != nil {
			return err
		}
		if call, ok := stmt.Expr.(*ast.CallExpr); ok && call.Package == "" && (call.Callee == "print" || call.Callee == "assert" || call.Callee == "delete") {
			return nil
		}
		if _, isTuple := typ.(*ast.TupleType); isTuple {
			return typeError(stmt.Pos(), "result of multi-value call must be assigned or discarded with _")
		}
		if ast.TypeEqual(typ, ast.ErrorType) {
			return typeError(stmt.Pos(), "result of call returning error must be handled or assigned to _")
		}
		return nil
	default:
		return typeError(stmt.Pos(), "unsupported statement %T", stmt)
	}
}

func (c *checker) checkKernelStmt(fn *ast.FuncDecl, locals *scope, stmt ast.Statement) error {
	switch stmt := stmt.(type) {
	case *ast.LetStmt:
		if _, exists := locals.locals[stmt.Name]; exists {
			return typeError(stmt.Start, "duplicate local variable %q", stmt.Name)
		}
		if !kernelLocalType(stmt.Type) {
			return typeError(stmt.Start, "kernel local %q has unsupported type %s", stmt.Name, stmt.Type)
		}
		valueType, err := c.checkKernelExpr(locals, stmt.Value)
		if err != nil {
			return err
		}
		if !ast.TypeEqual(valueType, stmt.Type) {
			return typeError(stmt.Value.Pos(), "cannot assign %s to %s", valueType, stmt.Type)
		}
		c.info.Locals[fn][stmt.Name] = stmt.Type
		locals.locals[stmt.Name] = localInfo{typ: stmt.Type, origin: OriginOwned}
		return nil
	case *ast.AssignStmt:
		local, ok := locals.lookup(stmt.Name)
		if !ok {
			return typeError(stmt.Start, "undefined variable %q", stmt.Name)
		}
		if _, isBuffer := local.typ.(*ast.GPUBufferType); isBuffer {
			return typeError(stmt.Start, "cannot assign gpu.Buffer variable %q", stmt.Name)
		}
		valueType, err := c.checkKernelExpr(locals, stmt.Value)
		if err != nil {
			return err
		}
		if !ast.TypeEqual(valueType, local.typ) {
			return typeError(stmt.Value.Pos(), "cannot assign %s to %s", valueType, local.typ)
		}
		return nil
	case *ast.IndexAssignStmt:
		collectionType, err := c.checkKernelExpr(locals, stmt.Target.Collection)
		if err != nil {
			return err
		}
		indexType, err := c.checkKernelExpr(locals, stmt.Target.Index)
		if err != nil {
			return err
		}
		if !ast.TypeEqual(indexType, ast.IntType) {
			return typeError(stmt.Target.Index.Pos(), "index must be int, got %s", indexType)
		}
		buf, ok := collectionType.(*ast.GPUBufferType)
		if !ok {
			return typeError(stmt.Target.Collection.Pos(), "kernel index assignment expects gpu.Buffer, got %s", collectionType)
		}
		valueType, err := c.checkKernelExpr(locals, stmt.Value)
		if err != nil {
			return err
		}
		if !ast.TypeEqual(valueType, buf.Elem) {
			return typeError(stmt.Value.Pos(), "cannot assign %s to %s element", valueType, buf.Elem)
		}
		c.setExpr(stmt.Target, buf.Elem, OriginOwned)
		return nil
	case *ast.IfStmt:
		conditionType, err := c.checkKernelExpr(locals, stmt.Condition)
		if err != nil {
			return err
		}
		if !ast.TypeEqual(conditionType, ast.BoolType) {
			return typeError(stmt.Condition.Pos(), "if condition must be bool, got %s", conditionType)
		}
		thenScope := newScope(locals)
		for _, inner := range stmt.Then.Statements {
			if err := c.checkKernelStmt(fn, thenScope, inner); err != nil {
				return err
			}
		}
		if stmt.Else != nil {
			elseScope := newScope(locals)
			for _, inner := range stmt.Else.Statements {
				if err := c.checkKernelStmt(fn, elseScope, inner); err != nil {
					return err
				}
			}
		}
		return nil
	case *ast.ForStmt:
		loopScope := newScope(locals)
		if stmt.Init != nil {
			if err := c.checkKernelStmt(fn, loopScope, stmt.Init); err != nil {
				return err
			}
		}
		if stmt.Condition != nil {
			conditionType, err := c.checkKernelExpr(loopScope, stmt.Condition)
			if err != nil {
				return err
			}
			if !ast.TypeEqual(conditionType, ast.BoolType) {
				return typeError(stmt.Condition.Pos(), "for condition must be bool, got %s", conditionType)
			}
		}
		c.loopDepth++
		bodyScope := newScope(loopScope)
		for _, inner := range stmt.Body.Statements {
			if err := c.checkKernelStmt(fn, bodyScope, inner); err != nil {
				c.loopDepth--
				return err
			}
		}
		if stmt.Post != nil {
			if err := c.checkKernelStmt(fn, loopScope, stmt.Post); err != nil {
				c.loopDepth--
				return err
			}
		}
		c.loopDepth--
		return nil
	case *ast.BreakStmt:
		if c.loopDepth == 0 {
			return typeError(stmt.Start, "break outside of a loop")
		}
		return nil
	case *ast.ContinueStmt:
		if c.loopDepth == 0 {
			return typeError(stmt.Start, "continue outside of a loop")
		}
		return nil
	case *ast.ReturnStmt:
		return typeError(stmt.Start, "kernel functions cannot return values")
	case *ast.ExprStmt:
		if call, ok := stmt.Expr.(*ast.CallExpr); ok && call.Package == "gpu" && (call.Callee == "sync" || call.Callee == "free" || call.Callee == "launch") {
			return typeError(stmt.Pos(), "gpu.%s cannot be used inside a kernel", call.Callee)
		}
		_, err := c.checkKernelExpr(locals, stmt.Expr)
		return err
	default:
		return typeError(stmt.Pos(), "unsupported kernel statement %T", stmt)
	}
}

func (c *checker) checkKernelExpr(locals *scope, expr ast.Expression) (ast.Type, error) {
	switch expr := expr.(type) {
	case *ast.IntLiteral:
		c.setExpr(expr, ast.IntType, OriginOwned)
		return ast.IntType, nil
	case *ast.FloatLiteral:
		c.setExpr(expr, ast.FloatType, OriginOwned)
		return ast.FloatType, nil
	case *ast.BoolLiteral:
		c.setExpr(expr, ast.BoolType, OriginOwned)
		return ast.BoolType, nil
	case *ast.NilLiteral:
		return nil, typeError(expr.Pos(), "nil is not supported in kernels")
	case *ast.IdentExpr:
		local, ok := locals.lookup(expr.Name)
		if !ok {
			return nil, typeError(expr.Start, "undefined variable %q", expr.Name)
		}
		c.setExpr(expr, local.typ, local.origin)
		return local.typ, nil
	case *ast.UnaryExpr:
		operandType, err := c.checkKernelExpr(locals, expr.Operand)
		if err != nil {
			return nil, err
		}
		typ, err := unaryType(expr.Start, expr.Operator, operandType)
		if err != nil {
			return nil, err
		}
		c.setExpr(expr, typ, OriginOwned)
		return typ, nil
	case *ast.BinaryExpr:
		leftType, err := c.checkKernelExpr(locals, expr.Left)
		if err != nil {
			return nil, err
		}
		rightType, err := c.checkKernelExpr(locals, expr.Right)
		if err != nil {
			return nil, err
		}
		typ, err := binaryType(expr.Start, expr.Operator, leftType, rightType)
		if err != nil {
			return nil, err
		}
		c.setExpr(expr, typ, OriginOwned)
		return typ, nil
	case *ast.IndexExpr:
		collectionType, err := c.checkKernelExpr(locals, expr.Collection)
		if err != nil {
			return nil, err
		}
		indexType, err := c.checkKernelExpr(locals, expr.Index)
		if err != nil {
			return nil, err
		}
		if !ast.TypeEqual(indexType, ast.IntType) {
			return nil, typeError(expr.Index.Pos(), "index must be int, got %s", indexType)
		}
		buf, ok := collectionType.(*ast.GPUBufferType)
		if !ok {
			return nil, typeError(expr.Collection.Pos(), "kernel indexing expects gpu.Buffer, got %s", collectionType)
		}
		c.setExpr(expr, buf.Elem, OriginOwned)
		return buf.Elem, nil
	case *ast.CallExpr:
		if expr.Package != "gpu" {
			return nil, typeError(expr.Start, "kernel cannot call %q", expr.SourceName())
		}
		typ, origin, err := c.checkGPUCall(locals, expr, false)
		if err != nil {
			return nil, err
		}
		switch c.info.GPUCalls[expr].Kind {
		case GPUCallGlobalX, GPUCallGlobalY, GPUCallGlobalZ, GPUCallThreadX, GPUCallThreadY, GPUCallThreadZ, GPUCallBlockX, GPUCallBlockY, GPUCallBlockZ, GPUCallBlockDimX, GPUCallBlockDimY, GPUCallBlockDimZ:
		default:
			return nil, typeError(expr.Start, "gpu.%s cannot be used inside a kernel", expr.Callee)
		}
		c.setExpr(expr, typ, origin)
		return typ, nil
	default:
		return nil, typeError(expr.Pos(), "unsupported kernel expression %T", expr)
	}
}

func (c *checker) checkLetValue(locals *scope, declared ast.Type, value ast.Expression) (ast.Type, error) {
	if call, ok := value.(*ast.CallExpr); ok && call.Package == "gpu" && call.Callee == "alloc" {
		buf, ok := declared.(*ast.GPUBufferType)
		if !ok {
			return nil, typeError(call.Start, "gpu.alloc result must be assigned to gpu.Buffer, got %s", declared)
		}
		if !gpuScalarType(buf.Elem) {
			return nil, typeError(call.Start, "gpu.Buffer element type must be int or float, got %s", buf.Elem)
		}
		if len(call.Args) != 1 {
			return nil, typeError(call.Start, "gpu.alloc expects 1 argument, got %d", len(call.Args))
		}
		lenType, err := c.checkExpr(locals, call.Args[0], false)
		if err != nil {
			return nil, err
		}
		if !ast.TypeEqual(lenType, ast.IntType) {
			return nil, typeError(call.Args[0].Pos(), "gpu.alloc length has type %s, want int", lenType)
		}
		c.info.GPUCalls[call] = GPUCallSig{Kind: GPUCallAlloc, BufferType: declared}
		c.setExpr(call, declared, OriginOwned)
		return declared, nil
	}
	return c.checkExpr(locals, value, false)
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
	case *ast.NilLiteral:
		c.setExpr(expr, ast.ErrorType, OriginOwned)
		return ast.ErrorType, nil
	case *ast.ArrayLiteral:
		if err := c.validateType(expr.Start, expr.Type); err != nil {
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
			elemType = c.coerceNil(elem, elemType, arrayType.Elem)
			if !ast.TypeEqual(elemType, arrayType.Elem) {
				return nil, typeError(elem.Pos(), "array element %d has type %s, want %s", i+1, elemType, arrayType.Elem)
			}
		}
		c.setExpr(expr, expr.Type, OriginScratch)
		return expr.Type, nil
	case *ast.ListLiteral:
		if err := c.validateType(expr.Start, expr.Type); err != nil {
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
			elemType = c.coerceNil(elem, elemType, listType.Elem)
			if !ast.TypeEqual(elemType, listType.Elem) {
				return nil, typeError(elem.Pos(), "list element %d has type %s, want %s", i+1, elemType, listType.Elem)
			}
		}
		c.setExpr(expr, expr.Type, OriginScratch)
		return expr.Type, nil
	case *ast.MapLiteral:
		if err := c.validateType(expr.Start, expr.Type); err != nil {
			return nil, err
		}
		for _, entry := range expr.Entries {
			keyType, err := c.checkExpr(locals, entry.Key, false)
			if err != nil {
				return nil, err
			}
			keyType = c.coerceNil(entry.Key, keyType, expr.Type.Key)
			if !ast.TypeEqual(keyType, expr.Type.Key) {
				return nil, typeError(entry.Key.Pos(), "map key has type %s, want %s", keyType, expr.Type.Key)
			}
			valType, err := c.checkExpr(locals, entry.Value, false)
			if err != nil {
				return nil, err
			}
			valType = c.coerceNil(entry.Value, valType, expr.Type.Value)
			if !ast.TypeEqual(valType, expr.Type.Value) {
				return nil, typeError(entry.Value.Pos(), "map value has type %s, want %s", valType, expr.Type.Value)
			}
		}
		c.setExpr(expr, expr.Type, OriginScratch)
		return expr.Type, nil
	case *ast.StructLiteral:
		if expr.Type.Package == "sqlite" {
			return nil, typeError(expr.Start, "cannot instantiate %s directly; use sqlite functions", expr.Type)
		}
		if expr.Type.Package == "json" {
			return nil, typeError(expr.Start, "cannot instantiate %s directly; use json functions", expr.Type)
		}
		if expr.Type.Package == "jwt" {
			return nil, typeError(expr.Start, "cannot instantiate %s directly; use jwt functions", expr.Type)
		}
		if err := c.validateType(expr.Start, expr.Type); err != nil {
			return nil, err
		}
		sig := c.info.Structs[expr.Type.TargetName()]
		seen := make(map[string]struct{}, len(expr.Fields))
		for _, init := range expr.Fields {
			fieldType, ok := sig.FieldMap[init.Name]
			if !ok {
				return nil, typeError(init.Pos, "struct %s has no field %q", sig.DisplayName, init.Name)
			}
			if _, dup := seen[init.Name]; dup {
				return nil, typeError(init.Pos, "duplicate field %q in struct literal", init.Name)
			}
			seen[init.Name] = struct{}{}
			valType, err := c.checkExpr(locals, init.Value, false)
			if err != nil {
				return nil, err
			}
			valType = c.coerceNil(init.Value, valType, fieldType)
			if !ast.TypeEqual(valType, fieldType) {
				return nil, typeError(init.Value.Pos(), "field %q in %s literal has type %s, want %s", init.Name, sig.DisplayName, valType, fieldType)
			}
		}
		origin := OriginOwned
		if c.isDynamicType(expr.Type) {
			origin = OriginScratch
		}
		c.setExpr(expr, expr.Type, origin)
		return expr.Type, nil
	case *ast.NewExpr:
		if err := c.validateType(expr.Start, expr.ElemType); err != nil {
			return nil, err
		}
		resType := &ast.PointerType{Elem: expr.ElemType}
		c.setExpr(expr, resType, OriginScratch)
		return resType, nil
	case *ast.MakeExpr:
		if err := c.validateType(expr.Start, expr.Type); err != nil {
			return nil, err
		}
		switch expr.Type.(type) {
		case *ast.SliceType:
			if expr.Len == nil {
				return nil, typeError(expr.Start, "make slice requires length argument")
			}
			lenType, err := c.checkExpr(locals, expr.Len, false)
			if err != nil {
				return nil, err
			}
			if !ast.TypeEqual(lenType, ast.IntType) {
				return nil, typeError(expr.Len.Pos(), "make length must be int, got %s", lenType)
			}
		case *ast.MapType:
			if expr.Len != nil {
				lenType, err := c.checkExpr(locals, expr.Len, false)
				if err != nil {
					return nil, err
				}
				if !ast.TypeEqual(lenType, ast.IntType) {
					return nil, typeError(expr.Len.Pos(), "make map capacity must be int, got %s", lenType)
				}
			}
		default:
			return nil, typeError(expr.Start, "make expects slice type, got %s", expr.Type)
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
	case *ast.FieldExpr:
		objType, err := c.checkExpr(locals, expr.Object, false)
		if err != nil {
			return nil, err
		}
		var structType *ast.StructType
		switch t := objType.(type) {
		case *ast.StructType:
			structType = t
		case *ast.PointerType:
			if st, ok := t.Elem.(*ast.StructType); ok {
				structType = st
			}
		}
		if structType == nil {
			return nil, typeError(expr.Object.Pos(), "cannot access field %q on non-struct type %s", expr.Field, objType)
		}
		sig, ok := c.info.Structs[structType.TargetName()]
		if !ok {
			return nil, typeError(expr.Object.Pos(), "undefined struct %q", structType)
		}
		fieldType, ok := sig.FieldMap[expr.Field]
		if !ok {
			return nil, typeError(expr.Pos(), "struct %s has no field %q", sig.DisplayName, expr.Field)
		}
		origin := OriginOwned
		if c.isDynamicType(fieldType) {
			origin = c.exprOrigin(expr.Object)
		}
		c.setExpr(expr, fieldType, origin)
		return fieldType, nil
	case *ast.UnaryExpr:
		operandType, err := c.checkExpr(locals, expr.Operand, false)
		if err != nil {
			return nil, err
		}
		switch expr.Operator {
		case "*":
			ptr, ok := operandType.(*ast.PointerType)
			if !ok {
				return nil, typeError(expr.Start, "cannot dereference non-pointer type %s", operandType)
			}
			c.setExpr(expr, ptr.Elem, OriginOwned)
			return ptr.Elem, nil
		case "&":
			_, isStructLit := expr.Operand.(*ast.StructLiteral)
			if !isStructLit && !c.isAddressable(expr.Operand) {
				return nil, typeError(expr.Operand.Pos(), "cannot take address of non-addressable expression")
			}
			resType := &ast.PointerType{Elem: operandType}
			origin := OriginOwned
			if isStructLit {
				origin = OriginScratch
			}
			c.setExpr(expr, resType, origin)
			return resType, nil
		}
		typ, err := unaryType(expr.Start, expr.Operator, operandType)
		if err != nil {
			return nil, err
		}
		c.setExpr(expr, typ, OriginOwned)
		return typ, nil
	case *ast.BinaryExpr:
		leftType, err := c.checkExpr(locals, expr.Left, false)
		if err != nil {
			return nil, err
		}
		rightType, err := c.checkExpr(locals, expr.Right, false)
		if err != nil {
			return nil, err
		}
		if expr.Operator == "==" || expr.Operator == "!=" {
			leftType = c.coerceNil(expr.Left, leftType, rightType)
			rightType = c.coerceNil(expr.Right, rightType, leftType)
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
		if mapType, isMap := collectionType.(*ast.MapType); isMap {
			indexType = c.coerceNil(expr.Index, indexType, mapType.Key)
			if !ast.TypeEqual(indexType, mapType.Key) {
				return nil, typeError(expr.Index.Pos(), "map key has type %s, want %s", indexType, mapType.Key)
			}
			c.setExpr(expr, mapType.Value, indexOrigin(mapType, c.exprOrigin(expr.Collection)))
			return mapType.Value, nil
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
	if expr.Package == "gpu" {
		typ, origin, err := c.checkGPUCall(locals, expr, allowPrint)
		if err != nil {
			return nil, err
		}
		c.setExpr(expr, typ, origin)
		return typ, nil
	}
	if expr.Package == "http" {
		typ, origin, err := c.checkHTTPCall(locals, expr, allowPrint)
		if err != nil {
			return nil, err
		}
		c.setExpr(expr, typ, origin)
		return typ, nil
	}

	argTypes := make([]ast.Type, 0, len(expr.Args))
	for _, arg := range expr.Args {
		argType, err := c.checkExpr(locals, arg, false)
		if err != nil {
			return nil, err
		}
		argTypes = append(argTypes, argType)
	}

	if expr.Package != "" && expr.ResolvedCallee == "" {
		if expr.Stdlib {
			member, ok := stdlib.LookupMember(expr.Package, expr.Callee)
			if !ok {
				return nil, typeError(expr.Start, "package %q has no function %q", expr.Package, expr.Callee)
			}
			typ, origin, err := c.checkStdlibCall(expr, member, argTypes, allowPrint)
			if err != nil {
				return nil, err
			}
			c.setExpr(expr, typ, origin)
			return typ, nil
		}
		return nil, typeError(expr.Start, "package %q is not imported", expr.Package)
	}

	if expr.Package == "" && expr.Callee == "print" {
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
	if expr.Package == "" && expr.Callee == "assert" {
		if !allowPrint {
			return nil, typeError(expr.Start, "assert can only be used as a statement")
		}
		if len(argTypes) != 2 {
			return nil, typeError(expr.Start, "assert expects 2 arguments, got %d", len(argTypes))
		}
		if !ast.TypeEqual(argTypes[0], ast.BoolType) {
			return nil, typeError(expr.Args[0].Pos(), "assert condition has type %s, want bool", argTypes[0])
		}
		if !ast.TypeEqual(argTypes[1], ast.StringType) {
			return nil, typeError(expr.Args[1].Pos(), "assert message has type %s, want string", argTypes[1])
		}
		c.info.AssertCalls[expr] = struct{}{}
		c.setExpr(expr, ast.StringType, OriginUnknown)
		return ast.StringType, nil
	}
	if expr.Package == "" && expr.Callee == "len" {
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
	if expr.Package == "" && expr.Callee == "clone" {
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
	if expr.Package == "" && expr.Callee == "append" {
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
		argTypes[1] = c.coerceNil(expr.Args[1], argTypes[1], listType.Elem)
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
	if expr.Package == "" && expr.Callee == "delete" {
		if !allowPrint {
			return nil, typeError(expr.Start, "delete can only be used as a statement")
		}
		if len(argTypes) != 2 {
			return nil, typeError(expr.Start, "delete expects 2 arguments, got %d", len(argTypes))
		}
		mapType, ok := argTypes[0].(*ast.MapType)
		if !ok {
			return nil, typeError(expr.Args[0].Pos(), "delete expects map as first argument, got %s", argTypes[0])
		}
		argTypes[1] = c.coerceNil(expr.Args[1], argTypes[1], mapType.Key)
		if !ast.TypeEqual(argTypes[1], mapType.Key) {
			return nil, typeError(expr.Args[1].Pos(), "delete key has type %s, want %s", argTypes[1], mapType.Key)
		}
		if err := c.checkCanMutate(expr.Args[0]); err != nil {
			return nil, err
		}
		c.info.DeleteCalls[expr] = mapType
		c.setExpr(expr, argTypes[0], c.exprOrigin(expr.Args[0]))
		return argTypes[0], nil
	}
	if expr.Package == "" && expr.Callee == "keys" {
		if len(argTypes) != 1 {
			return nil, typeError(expr.Start, "keys expects 1 argument, got %d", len(argTypes))
		}
		mapType, ok := argTypes[0].(*ast.MapType)
		if !ok {
			return nil, typeError(expr.Args[0].Pos(), "keys expects map argument, got %s", argTypes[0])
		}
		resType := &ast.ListType{Elem: mapType.Key}
		c.info.KeysCalls[expr] = mapType
		c.setExpr(expr, resType, OriginFrameOwned)
		return resType, nil
	}
	if expr.Package == "" && expr.Callee == "values" {
		if len(argTypes) != 1 {
			return nil, typeError(expr.Start, "values expects 1 argument, got %d", len(argTypes))
		}
		mapType, ok := argTypes[0].(*ast.MapType)
		if !ok {
			return nil, typeError(expr.Args[0].Pos(), "values expects map argument, got %s", argTypes[0])
		}
		resType := &ast.ListType{Elem: mapType.Value}
		c.info.ValuesCalls[expr] = mapType
		c.setExpr(expr, resType, OriginFrameOwned)
		return resType, nil
	}
	targetName := expr.TargetName()
	displayName := expr.SourceName()
	sig, ok := c.info.Funcs[targetName]
	if !ok {
		if expr.Package == "" {
			if member, ok := stdlib.LookupLegacy(expr.Callee); ok {
				return nil, typeError(expr.Start, "undefined function %q; use %s after import %q", expr.Callee, member.SourceName(), member.Package)
			}
		}
		return nil, typeError(expr.Start, "undefined function %q", displayName)
	}
	if len(argTypes) != len(sig.Params) {
		return nil, typeError(expr.Start, "%s expects %d arguments, got %d", displayName, len(sig.Params), len(argTypes))
	}
	for i, argType := range argTypes {
		want := sig.Params[i].Type
		argType = c.coerceNil(expr.Args[i], argType, want)
		if !ast.TypeEqual(argType, want) {
			return nil, typeError(expr.Args[i].Pos(), "argument %d to %s has type %s, want %s", i+1, displayName, argType, want)
		}
		if sig.Params[i].Mutable {
			if err := c.checkMutableArgument(sig.Params[i], expr.Args[i]); err != nil {
				return nil, err
			}
		}
	}

	c.info.ResolvedCalls[expr] = sig
	c.setExpr(expr, sig.ReturnType, originForCallResult(sig.ReturnType))
	return sig.ReturnType, nil
}

func (c *checker) checkHTTPHandlerArg(arg ast.Expression, caller string) (FuncSig, error) {
	handlerName := ""
	switch h := arg.(type) {
	case *ast.IdentExpr:
		handlerName = h.Name
	case *ast.FieldExpr:
		if h.ResolvedIdent != "" {
			handlerName = h.ResolvedIdent
		} else if ident, ok := h.Object.(*ast.IdentExpr); ok && ident.Package == "" {
			handlerName = ident.Name + "." + h.Field
		}
	}
	if handlerName == "" {
		return FuncSig{}, typeError(arg.Pos(), "%s argument must be a handler function name", caller)
	}
	handler, ok := c.info.Funcs[handlerName]
	if !ok {
		return FuncSig{}, typeError(arg.Pos(), "undefined handler %q", handlerName)
	}
	if len(handler.Params) != 1 || !ast.TypeEqual(handler.Params[0].Type, ast.IntType) || !ast.TypeEqual(handler.ReturnType, ast.IntType) {
		return FuncSig{}, typeError(arg.Pos(), "http handler %q must have signature func %s(request int) int", handler.Name, handler.Name)
	}
	return handler, nil
}

func (c *checker) checkHTTPCall(locals *scope, expr *ast.CallExpr, allowStatementOnly bool) (ast.Type, Origin, error) {
	switch expr.Callee {
	case "listen":
		if !allowStatementOnly {
			return nil, "", typeError(expr.Start, "http.listen can only be used as a statement")
		}
		if len(expr.Args) != 1 {
			return nil, "", typeError(expr.Start, "http.listen expects 1 argument, got %d", len(expr.Args))
		}
		portType, err := c.checkExpr(locals, expr.Args[0], false)
		if err != nil {
			return nil, "", err
		}
		if !ast.TypeEqual(portType, ast.IntType) {
			return nil, "", typeError(expr.Args[0].Pos(), "http.listen port has type %s, want int", portType)
		}
		c.info.HTTPCalls[expr] = HTTPCallSig{Kind: HTTPCallListen}
		return ast.StringType, OriginUnknown, nil
	case "serve":
		if !allowStatementOnly {
			return nil, "", typeError(expr.Start, "http.serve can only be used as a statement")
		}
		if len(expr.Args) != 3 {
			return nil, "", typeError(expr.Start, "http.serve expects 3 arguments, got %d", len(expr.Args))
		}
		hostType, err := c.checkExpr(locals, expr.Args[0], false)
		if err != nil {
			return nil, "", err
		}
		portType, err := c.checkExpr(locals, expr.Args[1], false)
		if err != nil {
			return nil, "", err
		}
		workersType, err := c.checkExpr(locals, expr.Args[2], false)
		if err != nil {
			return nil, "", err
		}
		if !ast.TypeEqual(hostType, ast.StringType) {
			return nil, "", typeError(expr.Args[0].Pos(), "http.serve host has type %s, want string", hostType)
		}
		if !ast.TypeEqual(portType, ast.IntType) {
			return nil, "", typeError(expr.Args[1].Pos(), "http.serve port has type %s, want int", portType)
		}
		if !ast.TypeEqual(workersType, ast.IntType) {
			return nil, "", typeError(expr.Args[2].Pos(), "http.serve workers has type %s, want int", workersType)
		}
		c.info.HTTPCalls[expr] = HTTPCallSig{Kind: HTTPCallServe}
		return ast.StringType, OriginUnknown, nil
	case "use":
		if !allowStatementOnly {
			return nil, "", typeError(expr.Start, "http.use can only be used as a statement")
		}
		if len(expr.Args) != 1 {
			return nil, "", typeError(expr.Start, "http.use expects 1 argument, got %d", len(expr.Args))
		}
		handler, err := c.checkHTTPHandlerArg(expr.Args[0], "http.use")
		if err != nil {
			return nil, "", err
		}
		c.info.HTTPCalls[expr] = HTTPCallSig{Kind: HTTPCallUse, HandlerName: handler.Name, HandlerSig: handler}
		return ast.StringType, OriginUnknown, nil
	case "useAt":
		if !allowStatementOnly {
			return nil, "", typeError(expr.Start, "http.useAt can only be used as a statement")
		}
		if len(expr.Args) != 2 {
			return nil, "", typeError(expr.Start, "http.useAt expects 2 arguments, got %d", len(expr.Args))
		}
		prefixType, err := c.checkExpr(locals, expr.Args[0], false)
		if err != nil {
			return nil, "", err
		}
		if !ast.TypeEqual(prefixType, ast.StringType) {
			return nil, "", typeError(expr.Args[0].Pos(), "http.useAt prefix has type %s, want string", prefixType)
		}
		handler, err := c.checkHTTPHandlerArg(expr.Args[1], "http.useAt")
		if err != nil {
			return nil, "", err
		}
		c.info.HTTPCalls[expr] = HTTPCallSig{Kind: HTTPCallUseAt, HandlerName: handler.Name, HandlerSig: handler}
		return ast.StringType, OriginUnknown, nil
	case "get", "post", "put", "delete", "patch":
		if !allowStatementOnly {
			return nil, "", typeError(expr.Start, "http.%s can only be used as a statement", expr.Callee)
		}
		if len(expr.Args) != 2 {
			return nil, "", typeError(expr.Start, "http.%s expects 2 arguments, got %d", expr.Callee, len(expr.Args))
		}
		pathType, err := c.checkExpr(locals, expr.Args[0], false)
		if err != nil {
			return nil, "", err
		}
		if !ast.TypeEqual(pathType, ast.StringType) {
			return nil, "", typeError(expr.Args[0].Pos(), "http.%s path has type %s, want string", expr.Callee, pathType)
		}
		handler, err := c.checkHTTPHandlerArg(expr.Args[1], "http."+expr.Callee)
		if err != nil {
			return nil, "", err
		}
		c.info.HTTPCalls[expr] = HTTPCallSig{Kind: HTTPCallKind(expr.Callee), HandlerName: handler.Name, HandlerSig: handler}
		return ast.StringType, OriginUnknown, nil
	case "route":
		if !allowStatementOnly {
			return nil, "", typeError(expr.Start, "http.route can only be used as a statement")
		}
		if len(expr.Args) != 3 {
			return nil, "", typeError(expr.Start, "http.route expects 3 arguments, got %d", len(expr.Args))
		}
		methodType, err := c.checkExpr(locals, expr.Args[0], false)
		if err != nil {
			return nil, "", err
		}
		pathType, err := c.checkExpr(locals, expr.Args[1], false)
		if err != nil {
			return nil, "", err
		}
		if !ast.TypeEqual(methodType, ast.StringType) {
			return nil, "", typeError(expr.Args[0].Pos(), "http.route method has type %s, want string", methodType)
		}
		if !ast.TypeEqual(pathType, ast.StringType) {
			return nil, "", typeError(expr.Args[1].Pos(), "http.route path has type %s, want string", pathType)
		}
		handler, err := c.checkHTTPHandlerArg(expr.Args[2], "http.route")
		if err != nil {
			return nil, "", err
		}
		c.info.HTTPCalls[expr] = HTTPCallSig{Kind: HTTPCallRoute, HandlerName: handler.Name, HandlerSig: handler}
		return ast.StringType, OriginUnknown, nil
	case "notFound":
		if !allowStatementOnly {
			return nil, "", typeError(expr.Start, "http.notFound can only be used as a statement")
		}
		if len(expr.Args) != 1 {
			return nil, "", typeError(expr.Start, "http.notFound expects 1 argument, got %d", len(expr.Args))
		}
		handler, err := c.checkHTTPHandlerArg(expr.Args[0], "http.notFound")
		if err != nil {
			return nil, "", err
		}
		c.info.HTTPCalls[expr] = HTTPCallSig{Kind: HTTPCallNotFound, HandlerName: handler.Name, HandlerSig: handler}
		return ast.StringType, OriginUnknown, nil
	case "param":
		if len(expr.Args) != 2 {
			return nil, "", typeError(expr.Start, "http.param expects 2 arguments, got %d", len(expr.Args))
		}
		reqType, err := c.checkExpr(locals, expr.Args[0], false)
		if err != nil {
			return nil, "", err
		}
		nameType, err := c.checkExpr(locals, expr.Args[1], false)
		if err != nil {
			return nil, "", err
		}
		if !ast.TypeEqual(reqType, ast.IntType) {
			return nil, "", typeError(expr.Args[0].Pos(), "http.param request has type %s, want int", reqType)
		}
		if !ast.TypeEqual(nameType, ast.StringType) {
			return nil, "", typeError(expr.Args[1].Pos(), "http.param name has type %s, want string", nameType)
		}
		c.info.HTTPCalls[expr] = HTTPCallSig{Kind: HTTPCallParam}
		return ast.StringType, OriginFrameOwned, nil
	case "set":
		if !allowStatementOnly {
			return nil, "", typeError(expr.Start, "http.set can only be used as a statement")
		}
		if len(expr.Args) != 3 {
			return nil, "", typeError(expr.Start, "http.set expects 3 arguments, got %d", len(expr.Args))
		}
		reqType, err := c.checkExpr(locals, expr.Args[0], false)
		if err != nil {
			return nil, "", err
		}
		keyType, err := c.checkExpr(locals, expr.Args[1], false)
		if err != nil {
			return nil, "", err
		}
		valType, err := c.checkExpr(locals, expr.Args[2], false)
		if err != nil {
			return nil, "", err
		}
		if !ast.TypeEqual(reqType, ast.IntType) {
			return nil, "", typeError(expr.Args[0].Pos(), "http.set request has type %s, want int", reqType)
		}
		if !ast.TypeEqual(keyType, ast.StringType) {
			return nil, "", typeError(expr.Args[1].Pos(), "http.set key has type %s, want string", keyType)
		}
		if !ast.TypeEqual(valType, ast.StringType) {
			return nil, "", typeError(expr.Args[2].Pos(), "http.set value has type %s, want string", valType)
		}
		c.info.HTTPCalls[expr] = HTTPCallSig{Kind: HTTPCallSet}
		return ast.StringType, OriginUnknown, nil
	case "context":
		if len(expr.Args) != 2 {
			return nil, "", typeError(expr.Start, "http.context expects 2 arguments, got %d", len(expr.Args))
		}
		reqType, err := c.checkExpr(locals, expr.Args[0], false)
		if err != nil {
			return nil, "", err
		}
		keyType, err := c.checkExpr(locals, expr.Args[1], false)
		if err != nil {
			return nil, "", err
		}
		if !ast.TypeEqual(reqType, ast.IntType) {
			return nil, "", typeError(expr.Args[0].Pos(), "http.context request has type %s, want int", reqType)
		}
		if !ast.TypeEqual(keyType, ast.StringType) {
			return nil, "", typeError(expr.Args[1].Pos(), "http.context key has type %s, want string", keyType)
		}
		c.info.HTTPCalls[expr] = HTTPCallSig{Kind: HTTPCallContext}
		return ast.StringType, OriginFrameOwned, nil
	case "setHeader":
		if !allowStatementOnly {
			return nil, "", typeError(expr.Start, "http.setHeader can only be used as a statement")
		}
		if len(expr.Args) != 3 {
			return nil, "", typeError(expr.Start, "http.setHeader expects 3 arguments, got %d", len(expr.Args))
		}
		reqType, err := c.checkExpr(locals, expr.Args[0], false)
		if err != nil {
			return nil, "", err
		}
		nameType, err := c.checkExpr(locals, expr.Args[1], false)
		if err != nil {
			return nil, "", err
		}
		valType, err := c.checkExpr(locals, expr.Args[2], false)
		if err != nil {
			return nil, "", err
		}
		if !ast.TypeEqual(reqType, ast.IntType) {
			return nil, "", typeError(expr.Args[0].Pos(), "http.setHeader request has type %s, want int", reqType)
		}
		if !ast.TypeEqual(nameType, ast.StringType) {
			return nil, "", typeError(expr.Args[1].Pos(), "http.setHeader name has type %s, want string", nameType)
		}
		if !ast.TypeEqual(valType, ast.StringType) {
			return nil, "", typeError(expr.Args[2].Pos(), "http.setHeader value has type %s, want string", valType)
		}
		c.info.HTTPCalls[expr] = HTTPCallSig{Kind: HTTPCallSetHeader}
		return ast.StringType, OriginUnknown, nil
	case "text", "json":
		if !allowStatementOnly {
			return nil, "", typeError(expr.Start, "http.%s can only be used as a statement", expr.Callee)
		}
		if len(expr.Args) != 3 {
			return nil, "", typeError(expr.Start, "http.%s expects 3 arguments, got %d", expr.Callee, len(expr.Args))
		}
		reqType, err := c.checkExpr(locals, expr.Args[0], false)
		if err != nil {
			return nil, "", err
		}
		statusType, err := c.checkExpr(locals, expr.Args[1], false)
		if err != nil {
			return nil, "", err
		}
		bodyType, err := c.checkExpr(locals, expr.Args[2], false)
		if err != nil {
			return nil, "", err
		}
		if !ast.TypeEqual(reqType, ast.IntType) {
			return nil, "", typeError(expr.Args[0].Pos(), "http.%s request has type %s, want int", expr.Callee, reqType)
		}
		if !ast.TypeEqual(statusType, ast.IntType) {
			return nil, "", typeError(expr.Args[1].Pos(), "http.%s status has type %s, want int", expr.Callee, statusType)
		}
		if !ast.TypeEqual(bodyType, ast.StringType) {
			return nil, "", typeError(expr.Args[2].Pos(), "http.%s body has type %s, want string", expr.Callee, bodyType)
		}
		kind := HTTPCallText
		if expr.Callee == "json" {
			kind = HTTPCallJSON
		}
		c.info.HTTPCalls[expr] = HTTPCallSig{Kind: kind}
		return ast.StringType, OriginUnknown, nil
	case "status":
		if !allowStatementOnly {
			return nil, "", typeError(expr.Start, "http.status can only be used as a statement")
		}
		if len(expr.Args) != 2 {
			return nil, "", typeError(expr.Start, "http.status expects 2 arguments, got %d", len(expr.Args))
		}
		reqType, err := c.checkExpr(locals, expr.Args[0], false)
		if err != nil {
			return nil, "", err
		}
		statusType, err := c.checkExpr(locals, expr.Args[1], false)
		if err != nil {
			return nil, "", err
		}
		if !ast.TypeEqual(reqType, ast.IntType) {
			return nil, "", typeError(expr.Args[0].Pos(), "http.status request has type %s, want int", reqType)
		}
		if !ast.TypeEqual(statusType, ast.IntType) {
			return nil, "", typeError(expr.Args[1].Pos(), "http.status status has type %s, want int", statusType)
		}
		c.info.HTTPCalls[expr] = HTTPCallSig{Kind: HTTPCallStatus}
		return ast.StringType, OriginUnknown, nil
	case "method", "path", "query", "body":
		if len(expr.Args) != 1 {
			return nil, "", typeError(expr.Start, "http.%s expects 1 argument, got %d", expr.Callee, len(expr.Args))
		}
		requestType, err := c.checkExpr(locals, expr.Args[0], false)
		if err != nil {
			return nil, "", err
		}
		if !ast.TypeEqual(requestType, ast.IntType) {
			return nil, "", typeError(expr.Args[0].Pos(), "http.%s request has type %s, want int", expr.Callee, requestType)
		}
		c.info.HTTPCalls[expr] = HTTPCallSig{Kind: HTTPCallKind(expr.Callee)}
		return ast.StringType, OriginFrameOwned, nil
	case "header":
		if len(expr.Args) != 2 {
			return nil, "", typeError(expr.Start, "http.header expects 2 arguments, got %d", len(expr.Args))
		}
		requestType, err := c.checkExpr(locals, expr.Args[0], false)
		if err != nil {
			return nil, "", err
		}
		nameType, err := c.checkExpr(locals, expr.Args[1], false)
		if err != nil {
			return nil, "", err
		}
		if !ast.TypeEqual(requestType, ast.IntType) {
			return nil, "", typeError(expr.Args[0].Pos(), "http.header request has type %s, want int", requestType)
		}
		if !ast.TypeEqual(nameType, ast.StringType) {
			return nil, "", typeError(expr.Args[1].Pos(), "http.header name has type %s, want string", nameType)
		}
		c.info.HTTPCalls[expr] = HTTPCallSig{Kind: HTTPCallHeader}
		return ast.StringType, OriginFrameOwned, nil
	case "respond":
		if !allowStatementOnly {
			return nil, "", typeError(expr.Start, "http.respond can only be used as a statement")
		}
		if len(expr.Args) != 4 {
			return nil, "", typeError(expr.Start, "http.respond expects 4 arguments, got %d", len(expr.Args))
		}
		argTypes := make([]ast.Type, 0, len(expr.Args))
		for _, arg := range expr.Args {
			argType, err := c.checkExpr(locals, arg, false)
			if err != nil {
				return nil, "", err
			}
			argTypes = append(argTypes, argType)
		}
		wants := []ast.Type{ast.IntType, ast.IntType, ast.StringType, ast.StringType}
		names := []string{"request", "status", "contentType", "body"}
		for i, want := range wants {
			if !ast.TypeEqual(argTypes[i], want) {
				return nil, "", typeError(expr.Args[i].Pos(), "http.respond %s has type %s, want %s", names[i], argTypes[i], want)
			}
		}
		c.info.HTTPCalls[expr] = HTTPCallSig{Kind: HTTPCallRespond}
		return ast.StringType, OriginUnknown, nil
	default:
		return nil, "", typeError(expr.Start, "package %q has no function %q", expr.Package, expr.Callee)
	}
}

func (c *checker) checkGPUCall(locals *scope, expr *ast.CallExpr, allowStatementOnly bool) (ast.Type, Origin, error) {
	switch expr.Callee {
	case "alloc":
		return nil, "", typeError(expr.Start, "gpu.alloc must be assigned to an explicitly typed gpu.Buffer")
	case "from":
		return c.checkGPUFrom(locals, expr)
	case "copyToDevice", "copyToHost", "to":
		if !allowStatementOnly {
			return nil, "", typeError(expr.Start, "gpu.%s can only be used as a statement", expr.Callee)
		}
		if len(expr.Args) != 2 {
			return nil, "", typeError(expr.Start, "gpu.%s expects 2 arguments, got %d", expr.Callee, len(expr.Args))
		}
		left, err := c.checkExpr(locals, expr.Args[0], false)
		if err != nil {
			return nil, "", err
		}
		right, err := c.checkExpr(locals, expr.Args[1], false)
		if err != nil {
			return nil, "", err
		}
		kind := GPUCallKind(expr.Callee)
		host, device := left, right
		hostExpr, deviceExpr := expr.Args[0], expr.Args[1]
		if expr.Callee != "copyToDevice" {
			if expr.Callee == "to" {
				kind = GPUCallCopyToHost
			}
			device, host = left, right
			deviceExpr, hostExpr = expr.Args[0], expr.Args[1]
		}
		hostSlice, ok := host.(*ast.SliceType)
		if !ok {
			return nil, "", typeError(hostExpr.Pos(), "gpu.%s host argument must be slice, got %s", expr.Callee, host)
		}
		deviceBuffer, ok := device.(*ast.GPUBufferType)
		if !ok {
			return nil, "", typeError(deviceExpr.Pos(), "gpu.%s device argument must be gpu.Buffer, got %s", expr.Callee, device)
		}
		if !ast.TypeEqual(hostSlice.Elem, deviceBuffer.Elem) {
			return nil, "", typeError(expr.Start, "gpu.%s element type mismatch: host has %s, device has %s", expr.Callee, hostSlice.Elem, deviceBuffer.Elem)
		}
		c.info.GPUCalls[expr] = GPUCallSig{Kind: kind}
		return ast.StringType, OriginUnknown, nil
	case "free", "sync":
		if !allowStatementOnly {
			return nil, "", typeError(expr.Start, "gpu.%s can only be used as a statement", expr.Callee)
		}
		if expr.Callee == "sync" {
			if len(expr.Args) != 0 {
				return nil, "", typeError(expr.Start, "gpu.sync expects 0 arguments, got %d", len(expr.Args))
			}
			c.info.GPUCalls[expr] = GPUCallSig{Kind: GPUCallSync}
			return ast.StringType, OriginUnknown, nil
		}
		if len(expr.Args) != 1 {
			return nil, "", typeError(expr.Start, "gpu.free expects 1 argument, got %d", len(expr.Args))
		}
		argType, err := c.checkExpr(locals, expr.Args[0], false)
		if err != nil {
			return nil, "", err
		}
		if _, ok := argType.(*ast.GPUBufferType); !ok {
			return nil, "", typeError(expr.Args[0].Pos(), "gpu.free expects gpu.Buffer, got %s", argType)
		}
		c.info.GPUCalls[expr] = GPUCallSig{Kind: GPUCallFree}
		return ast.StringType, OriginUnknown, nil
	case "launch":
		return c.checkGPULaunch(locals, expr, allowStatementOnly, 0)
	case "launch1d":
		return c.checkGPULaunch(locals, expr, allowStatementOnly, 1)
	case "launch2d":
		return c.checkGPULaunch(locals, expr, allowStatementOnly, 2)
	case "globalX", "globalY", "globalZ", "threadX", "threadY", "threadZ", "blockX", "blockY", "blockZ", "blockDimX", "blockDimY", "blockDimZ":
		if len(expr.Args) != 0 {
			return nil, "", typeError(expr.Start, "gpu.%s expects 0 arguments, got %d", expr.Callee, len(expr.Args))
		}
		c.info.GPUCalls[expr] = GPUCallSig{Kind: GPUCallKind(expr.Callee)}
		return ast.IntType, OriginOwned, nil
	default:
		return nil, "", typeError(expr.Start, "package %q has no function %q", expr.Package, expr.Callee)
	}
}

func (c *checker) checkGPUFrom(locals *scope, expr *ast.CallExpr) (ast.Type, Origin, error) {
	if len(expr.Args) != 1 {
		return nil, "", typeError(expr.Start, "gpu.from expects 1 argument, got %d", len(expr.Args))
	}
	hostType, err := c.checkExpr(locals, expr.Args[0], false)
	if err != nil {
		return nil, "", err
	}
	hostSlice, ok := hostType.(*ast.SliceType)
	if !ok {
		return nil, "", typeError(expr.Args[0].Pos(), "gpu.from expects a slice, got %s", hostType)
	}
	if !gpuScalarType(hostSlice.Elem) {
		return nil, "", typeError(expr.Args[0].Pos(), "gpu.from element type must be int or float, got %s", hostSlice.Elem)
	}
	bufferType := &ast.GPUBufferType{Elem: hostSlice.Elem}
	c.info.GPUCalls[expr] = GPUCallSig{Kind: GPUCallFrom, BufferType: bufferType}
	return bufferType, OriginOwned, nil
}

func (c *checker) checkGPULaunch(locals *scope, expr *ast.CallExpr, allowStatementOnly bool, axes int) (ast.Type, Origin, error) {
	launchName := expr.Callee
	if !allowStatementOnly {
		return nil, "", typeError(expr.Start, "gpu.%s can only be used as a statement", launchName)
	}
	dimCount := 6
	if axes > 0 {
		dimCount = axes
	}
	if len(expr.Args) < 1+dimCount {
		if axes == 0 {
			return nil, "", typeError(expr.Start, "gpu.launch expects kernel, 6 launch dimensions, and kernel args")
		}
		return nil, "", typeError(expr.Start, "gpu.%s expects a kernel, %d extents, and kernel arguments", launchName, dimCount)
	}
	kernelIdent, ok := expr.Args[0].(*ast.IdentExpr)
	if !ok {
		return nil, "", typeError(expr.Args[0].Pos(), "gpu.%s first argument must be a kernel name", launchName)
	}
	kernel, ok := c.info.Kernels[kernelIdent.Name]
	if !ok {
		return nil, "", typeError(expr.Args[0].Pos(), "undefined kernel %q", kernelIdent.Name)
	}
	for i := 1; i <= dimCount; i++ {
		argType, err := c.checkExpr(locals, expr.Args[i], false)
		if err != nil {
			return nil, "", err
		}
		if !ast.TypeEqual(argType, ast.IntType) {
			return nil, "", typeError(expr.Args[i].Pos(), "gpu.%s extent %d has type %s, want int", launchName, i, argType)
		}
	}
	if axes > 0 {
		if len(kernel.Params) < axes {
			return nil, "", typeError(expr.Start, "gpu.%s kernel %s needs %d int coordinate parameters before its other arguments", launchName, kernel.Name, axes)
		}
		for i := 0; i < axes; i++ {
			if !ast.TypeEqual(kernel.Params[i].Type, ast.IntType) {
				return nil, "", typeError(expr.Start, "gpu.%s kernel %s coordinate parameter %q must be int, got %s", launchName, kernel.Name, kernel.Params[i].Name, kernel.Params[i].Type)
			}
		}
	}
	if prev, seen := c.info.KernelAxes[kernel.Name]; seen && prev != axes {
		return nil, "", typeError(expr.Start, "kernel %q is launched with two different shapes", kernel.Name)
	}
	c.info.KernelAxes[kernel.Name] = axes

	kernelArgs := expr.Args[1+dimCount:]
	wantParams := kernel.Params
	if axes > 0 {
		wantParams = kernel.Params[axes:]
	}
	if len(kernelArgs) != len(wantParams) {
		return nil, "", typeError(expr.Start, "gpu.%s kernel %s expects %d arguments, got %d", launchName, kernel.Name, len(wantParams), len(kernelArgs))
	}
	for i, arg := range kernelArgs {
		argType, err := c.checkExpr(locals, arg, false)
		if err != nil {
			return nil, "", err
		}
		want := wantParams[i].Type
		if !ast.TypeEqual(argType, want) {
			return nil, "", typeError(arg.Pos(), "argument %d to kernel %s has type %s, want %s", i+1, kernel.Name, argType, want)
		}
	}
	kind := GPUCallLaunch
	if axes == 1 {
		kind = GPUCallLaunch1D
	} else if axes == 2 {
		kind = GPUCallLaunch2D
	}
	c.info.GPUCalls[expr] = GPUCallSig{Kind: kind, KernelName: kernel.Name, KernelSig: kernel}
	return ast.StringType, OriginUnknown, nil
}

func (c *checker) checkStdlibCall(expr *ast.CallExpr, member stdlib.Member, argTypes []ast.Type, allowStatementOnly bool) (ast.Type, Origin, error) {
	displayName := member.SourceName()
	if member.StatementOnly && !allowStatementOnly {
		return nil, "", typeError(expr.Start, "%s can only be used as a statement", displayName)
	}
	if len(argTypes) != len(member.Params) {
		return nil, "", typeError(expr.Start, "%s expects %d arguments, got %d", displayName, len(member.Params), len(argTypes))
	}
	for i, argType := range argTypes {
		want := member.Params[i].Type
		if !ast.TypeEqual(argType, want) {
			return nil, "", typeError(expr.Args[i].Pos(), "%s %s has type %s, want %s", displayName, member.Params[i].Name, argType, want)
		}
	}

	if member.Package == "sqlite" {
		c.info.UsesSQLite = true
		c.info.SQLiteCalls[expr] = SQLiteCallSig{Kind: SQLiteCallKind(member.Kind)}
	} else if member.Package == "json" {
		c.info.UsesJSON = true
		c.info.JSONCalls[expr] = JSONCallSig{Kind: JSONCallKind(member.Kind)}
	} else if member.Package == "jwt" {
		c.info.UsesJWT = true
		c.info.JWTCalls[expr] = JWTCallSig{Kind: JWTCallKind(member.Kind)}
	} else {
		c.info.IOCalls[expr] = IOCallSig{Kind: IOCallKind(member.Kind)}
		if member.Kind == stdlib.CallEnv {
			c.info.UsesEnv = true
		}
	}
	return member.ReturnType, originForStdlibResult(member.ResultOrigin), nil
}

func originForStdlibResult(origin stdlib.ResultOrigin) Origin {
	switch origin {
	case stdlib.ResultOwned:
		return OriginOwned
	case stdlib.ResultFrameOwned:
		return OriginFrameOwned
	default:
		return OriginUnknown
	}
}

func unaryType(pos token.Position, operator string, operand ast.Type) (ast.Type, error) {
	switch operator {
	case "!":
		if ast.TypeEqual(operand, ast.BoolType) {
			return ast.BoolType, nil
		}
		return nil, typeError(pos, "operator %q requires a bool operand, got %s", operator, operand)
	case "-":
		if ast.TypeEqual(operand, ast.IntType) || ast.TypeEqual(operand, ast.FloatType) {
			return operand, nil
		}
		return nil, typeError(pos, "operator %q requires an int or float operand, got %s", operator, operand)
	default:
		return nil, typeError(pos, "unsupported operator %q", operator)
	}
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
	case "&&", "||":
		if ast.TypeEqual(leftType, ast.BoolType) && ast.TypeEqual(rightType, ast.BoolType) {
			return ast.BoolType, nil
		}
		return nil, typeError(pos, "operator %q requires bool operands, got %s and %s", operator, leftType, rightType)
	case "%":
		if ast.TypeEqual(leftType, ast.IntType) && ast.TypeEqual(rightType, ast.IntType) {
			return ast.IntType, nil
		}
		return nil, typeError(pos, "operator %q requires int operands, got %s and %s", operator, leftType, rightType)
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
	if _, isPtr := typ.(*ast.PointerType); isPtr {
		return true
	}
	return printableType(typ)
}

func printableType(typ ast.Type) bool {
	return ast.TypeEqual(typ, ast.IntType) ||
		ast.TypeEqual(typ, ast.FloatType) ||
		ast.TypeEqual(typ, ast.StringType) ||
		ast.TypeEqual(typ, ast.BoolType) ||
		ast.TypeEqual(typ, ast.ErrorType)
}

func hasLength(typ ast.Type) bool {
	if ast.TypeEqual(typ, ast.StringType) {
		return true
	}
	if _, isMap := typ.(*ast.MapType); isMap {
		return true
	}
	_, ok := ast.ElementType(typ)
	return ok
}

func isMutableCollection(typ ast.Type) bool {
	if _, isMap := typ.(*ast.MapType); isMap {
		return true
	}
	_, ok := ast.ElementType(typ)
	return ok
}

func cloneableType(typ ast.Type) bool {
	switch typ.(type) {
	case *ast.StructType, *ast.PointerType, *ast.MapType:
		return true
	}
	return ast.TypeEqual(typ, ast.StringType) || isMutableCollection(typ)
}

func (c *checker) isDynamicType(typ ast.Type) bool {
	if dynamicType(typ) {
		return true
	}
	if st, ok := typ.(*ast.StructType); ok {
		if sig, exists := c.info.Structs[st.TargetName()]; exists {
			for _, field := range sig.Fields {
				if c.isDynamicType(field.Type) {
					return true
				}
			}
		}
	}
	return false
}

func dynamicType(typ ast.Type) bool {
	if ast.TypeEqual(typ, ast.StringType) || ast.TypeEqual(typ, ast.ErrorType) || isMutableCollection(typ) {
		return true
	}
	if _, isPtr := typ.(*ast.PointerType); isPtr {
		return true
	}
	if tuple, ok := typ.(*ast.TupleType); ok {
		for _, t := range tuple.Types {
			if dynamicType(t) {
				return true
			}
		}
	}
	return false
}

func validateMutableParam(fn *ast.FuncDecl, param ast.Param) error {
	if !param.Mutable {
		return nil
	}
	if fn.Kernel {
		return typeError(param.Pos, "kernel parameter %q cannot be mut", param.Name)
	}
	if !isMutableCollection(param.Type) {
		return typeError(param.Pos, "mut parameter %q must be array, slice, or list, got %s", param.Name, param.Type)
	}
	return nil
}

func originForParam(param ast.Param) Origin {
	if param.Mutable {
		return OriginOwned
	}
	if _, isPtr := param.Type.(*ast.PointerType); isPtr {
		return OriginOwned
	}
	if dynamicType(param.Type) {
		return OriginBorrowed
	}
	return OriginOwned
}

// originForUnpackedValue is the origin of one value taken out of a call's
// tuple. Dynamic values are allocated in this function's frame, so returning
// them later must copy them into the caller. Marking them owned skipped that
// copy and left the caller with a pointer into a freed frame.
func originForUnpackedValue(typ ast.Type) Origin {
	if dynamicType(typ) {
		return OriginFrameOwned
	}
	return OriginOwned
}

func originForCallResult(typ ast.Type) Origin {
	if _, isPtr := typ.(*ast.PointerType); isPtr {
		return OriginOwned
	}
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
	if mapType, ok := collectionType.(*ast.MapType); ok {
		if dynamicType(mapType.Value) {
			return collectionOrigin
		}
		return OriginOwned
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

func (c *checker) checkMutableArgument(param ast.Param, arg ast.Expression) error {
	switch c.exprOrigin(arg) {
	case OriginBorrowed:
		return typeError(arg.Pos(), "cannot pass borrowed collection to mut parameter %q", param.Name)
	case OriginUnknown:
		return typeError(arg.Pos(), "cannot pass collection with unknown ownership to mut parameter %q", param.Name)
	default:
		return nil
	}
}

func validMapKeyType(typ ast.Type) bool {
	if ast.TypeEqual(typ, ast.IntType) || ast.TypeEqual(typ, ast.StringType) || ast.TypeEqual(typ, ast.BoolType) {
		return true
	}
	_, isPtr := typ.(*ast.PointerType)
	return isPtr
}

func (c *checker) validateType(pos token.Position, typ ast.Type) error {
	switch typ := typ.(type) {
	case ast.ScalarType:
		return nil
	case *ast.ArrayType:
		if typ.Length <= 0 {
			return typeError(pos, "array length must be positive")
		}
		return c.validateType(pos, typ.Elem)
	case *ast.SliceType:
		return c.validateType(pos, typ.Elem)
	case *ast.ListType:
		return c.validateType(pos, typ.Elem)
	case *ast.StructType:
		if typ.Package == "sqlite" && (typ.Name == "DB" || typ.Name == "Stmt") {
			c.info.UsesSQLite = true
			c.registerSQLiteStructs()
			targetName := "trux_sqlite_" + strings.ToLower(typ.Name)
			typ.ResolvedName = targetName
		}
		if typ.Package == "json" && (typ.Name == "Doc" || typ.Name == "Val" || typ.Name == "MutDoc" || typ.Name == "MutVal") {
			c.info.UsesJSON = true
			c.registerJSONStructs()
			var targetName string
			switch typ.Name {
			case "Doc":
				targetName = "trux_json_doc"
			case "Val":
				targetName = "trux_json_val"
			case "MutDoc":
				targetName = "trux_json_mut_doc"
			case "MutVal":
				targetName = "trux_json_mut_val"
			}
			typ.ResolvedName = targetName
		}
		if typ.Package == "jwt" && typ.Name == "Claims" {
			c.info.UsesJWT = true
			c.registerJWTStructs()
			typ.ResolvedName = "trux_jwt_claims"
		}
		sig, ok := c.info.Structs[typ.TargetName()]
		if !ok {
			return typeError(pos, "undefined struct %q", typ.String())
		}
		typ.ResolvedName = sig.Name
		return nil
	case *ast.PointerType:
		return c.validateType(pos, typ.Elem)
	case *ast.MapType:
		if err := c.validateType(pos, typ.Key); err != nil {
			return err
		}
		if !validMapKeyType(typ.Key) {
			return typeError(pos, "invalid map key type %s; map keys must be int, string, bool, or pointer", typ.Key)
		}
		return c.validateType(pos, typ.Value)
	case *ast.GPUBufferType:
		if !gpuScalarType(typ.Elem) {
			return typeError(pos, "gpu.Buffer element type must be int or float, got %s", typ.Elem)
		}
		return nil
	case *ast.TupleType:
		return typeError(pos, "tuple types can only be used as function return types")
	default:
		return typeError(pos, "unsupported type %s", typ)
	}
}

func (c *checker) validateReturnType(pos token.Position, typ ast.Type) error {
	if typ == nil {
		return nil
	}
	if tuple, ok := typ.(*ast.TupleType); ok {
		if len(tuple.Types) < 2 {
			return typeError(pos, "tuple type must have at least 2 types")
		}
		for _, elem := range tuple.Types {
			if _, isTuple := elem.(*ast.TupleType); isTuple {
				return typeError(pos, "nested tuple types are not supported")
			}
			if err := c.validateType(pos, elem); err != nil {
				return err
			}
		}
		return nil
	}
	return c.validateType(pos, typ)
}

func gpuScalarType(typ ast.Type) bool {
	return ast.TypeEqual(typ, ast.IntType) || ast.TypeEqual(typ, ast.FloatType)
}

func kernelLocalType(typ ast.Type) bool {
	return ast.TypeEqual(typ, ast.IntType) || ast.TypeEqual(typ, ast.FloatType) || ast.TypeEqual(typ, ast.BoolType)
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
