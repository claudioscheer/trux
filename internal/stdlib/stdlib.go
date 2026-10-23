package stdlib

import (
	"sort"
	"strings"

	"github.com/claudioscheer/trux/internal/ast"
)

type CallKind string

const (
	CallReadLine              CallKind = "readLine"
	CallReadInt               CallKind = "readInt"
	CallReadFloat             CallKind = "readFloat"
	CallReadBool              CallKind = "readBool"
	CallReadFile              CallKind = "readFile"
	CallWriteFile             CallKind = "writeFile"
	CallCreateDir             CallKind = "createDir"
	CallRemove                CallKind = "remove"
	CallReadDir               CallKind = "readDir"
	CallExists                CallKind = "exists"
	CallIsFile                CallKind = "isFile"
	CallIsDir                 CallKind = "isDir"
	CallEnv                   CallKind = "env"
	CallReadCSV               CallKind = "readCsv"
	CallWriteCSV              CallKind = "writeCsv"
	CallImageWidth            CallKind = "imageWidth"
	CallImageHeight           CallKind = "imageHeight"
	CallReadPPM               CallKind = "readPpm"
	CallWritePPM              CallKind = "writePpm"
	CallTimeNowUnixMillis     CallKind = "nowUnixMillis"
	CallTimeMonotonicNanos    CallKind = "monotonicNanos"
	CallTimeSleepMillis       CallKind = "sleepMillis"
	CallHTTPListen            CallKind = "listen"
	CallHTTPServe             CallKind = "serve"
	CallHTTPUse               CallKind = "use"
	CallHTTPUseAt             CallKind = "useAt"
	CallHTTPGet               CallKind = "get"
	CallHTTPPost              CallKind = "post"
	CallHTTPPut               CallKind = "put"
	CallHTTPDelete            CallKind = "delete"
	CallHTTPPatch             CallKind = "patch"
	CallHTTPRoute             CallKind = "route"
	CallHTTPNotFound          CallKind = "notFound"
	CallHTTPParam             CallKind = "param"
	CallHTTPSet               CallKind = "set"
	CallHTTPContext           CallKind = "context"
	CallHTTPSetHeader         CallKind = "setHeader"
	CallHTTPText              CallKind = "text"
	CallHTTPJSON              CallKind = "json"
	CallHTTPStatus            CallKind = "status"
	CallHTTPMethod            CallKind = "method"
	CallHTTPPath              CallKind = "path"
	CallHTTPQuery             CallKind = "query"
	CallHTTPHeader            CallKind = "header"
	CallHTTPBody              CallKind = "body"
	CallHTTPRespond           CallKind = "respond"
	CallGPUAlloc              CallKind = "gpuAlloc"
	CallGPUCopyToDevice       CallKind = "copyToDevice"
	CallGPUCopyToHost         CallKind = "copyToHost"
	CallGPUFree               CallKind = "gpuFree"
	CallGPUSync               CallKind = "gpuSync"
	CallGPULaunch             CallKind = "gpuLaunch"
	CallGPULaunch1D           CallKind = "gpuLaunch1d"
	CallGPULaunch2D           CallKind = "gpuLaunch2d"
	CallGPUFrom               CallKind = "gpuFrom"
	CallGPUMount              CallKind = "gpuMount"
	CallGPUWorkspaceFloat     CallKind = "gpuWorkspaceFloat"
	CallGPUWorkspaceInt       CallKind = "gpuWorkspaceInt"
	CallGPUFloats             CallKind = "gpuFloats"
	CallGPUInts               CallKind = "gpuInts"
	CallGPUDim                CallKind = "gpuDim"
	CallGPUScalar             CallKind = "gpuScalar"
	CallGPUHostFloats         CallKind = "gpuHostFloats"
	CallGPUHostInts           CallKind = "gpuHostInts"
	CallGPUBorrowFloat        CallKind = "gpuBorrowFloat"
	CallGPUBorrowInt          CallKind = "gpuBorrowInt"
	CallGPUReady              CallKind = "gpuReady"
	CallGPUFinish             CallKind = "gpuFinish"
	CallGPUTo                 CallKind = "gpuTo"
	CallGPUCoord              CallKind = "gpuCoord"
	CallErrorsNew             CallKind = "errorsNew"
	CallErrorsMessage         CallKind = "errorsMessage"
	CallSQLiteOpen            CallKind = "sqliteOpen"
	CallSQLiteOpenThreadLocal CallKind = "sqliteOpenThreadLocal"
	CallSQLiteClose           CallKind = "sqliteClose"
	CallSQLiteExec            CallKind = "sqliteExec"
	CallSQLiteQuery           CallKind = "sqliteQuery"
	CallSQLiteLastInsertRowID CallKind = "sqliteLastInsertRowId"
	CallSQLiteChanges         CallKind = "sqliteChanges"
	CallSQLiteTotalChanges    CallKind = "sqliteTotalChanges"
	CallSQLiteBegin           CallKind = "sqliteBegin"
	CallSQLiteCommit          CallKind = "sqliteCommit"
	CallSQLiteRollback        CallKind = "sqliteRollback"
	CallSQLitePrepare         CallKind = "sqlitePrepare"
	CallSQLiteStep            CallKind = "sqliteStep"
	CallSQLiteReset           CallKind = "sqliteReset"
	CallSQLiteClearBindings   CallKind = "sqliteClearBindings"
	CallSQLiteFinalize        CallKind = "sqliteFinalize"
	CallSQLiteBindInt         CallKind = "sqliteBindInt"
	CallSQLiteBindFloat       CallKind = "sqliteBindFloat"
	CallSQLiteBindText        CallKind = "sqliteBindText"
	CallSQLiteBindNull        CallKind = "sqliteBindNull"
	CallSQLiteBindIndex       CallKind = "sqliteBindIndex"
	CallSQLiteColumnCount     CallKind = "sqliteColumnCount"
	CallSQLiteColumnName      CallKind = "sqliteColumnName"
	CallSQLiteColumnIndex     CallKind = "sqliteColumnIndex"
	CallSQLiteColumnType      CallKind = "sqliteColumnType"
	CallSQLiteColumnInt       CallKind = "sqliteColumnInt"
	CallSQLiteColumnFloat     CallKind = "sqliteColumnFloat"
	CallSQLiteColumnText      CallKind = "sqliteColumnText"
	CallSQLiteColumnIsNull    CallKind = "sqliteColumnIsNull"
	CallJSONParse             CallKind = "jsonParse"
	CallJSONRoot              CallKind = "jsonRoot"
	CallJSONGet               CallKind = "jsonGet"
	CallJSONAt                CallKind = "jsonAt"
	CallJSONLen               CallKind = "jsonLen"
	CallJSONIsObject          CallKind = "jsonIsObject"
	CallJSONIsArray           CallKind = "jsonIsArray"
	CallJSONIsString          CallKind = "jsonIsString"
	CallJSONIsInt             CallKind = "jsonIsInt"
	CallJSONIsFloat           CallKind = "jsonIsFloat"
	CallJSONIsBool            CallKind = "jsonIsBool"
	CallJSONIsNull            CallKind = "jsonIsNull"
	CallJSONGetString         CallKind = "jsonGetString"
	CallJSONGetInt            CallKind = "jsonGetInt"
	CallJSONGetFloat          CallKind = "jsonGetFloat"
	CallJSONGetBool           CallKind = "jsonGetBool"
	CallJSONFieldString       CallKind = "jsonFieldString"
	CallJSONFieldInt          CallKind = "jsonFieldInt"
	CallJSONFieldFloat        CallKind = "jsonFieldFloat"
	CallJSONFieldBool         CallKind = "jsonFieldBool"
	CallJSONDecodeMap         CallKind = "jsonDecodeMap"
	CallJSONEncodeMap         CallKind = "jsonEncodeMap"
	CallJSONStringify         CallKind = "jsonStringify"
	CallJSONValid             CallKind = "jsonValid"
	CallJSONNewDoc            CallKind = "jsonNewDoc"
	CallJSONNewObject         CallKind = "jsonNewObject"
	CallJSONNewArray          CallKind = "jsonNewArray"
	CallJSONSetRoot           CallKind = "jsonSetRoot"
	CallJSONAddString         CallKind = "jsonAddString"
	CallJSONAddInt            CallKind = "jsonAddInt"
	CallJSONAddFloat          CallKind = "jsonAddFloat"
	CallJSONAddBool           CallKind = "jsonAddBool"
	CallJSONAddNull           CallKind = "jsonAddNull"
	CallJSONAddVal            CallKind = "jsonAddVal"
	CallJSONAppendVal         CallKind = "jsonAppendVal"
	CallJSONWrite             CallKind = "jsonWrite"
	CallJWTSign               CallKind = "jwtSign"
	CallJWTVerify             CallKind = "jwtVerify"
	CallJWTGetString          CallKind = "jwtGetString"
	CallJWTGetInt             CallKind = "jwtGetInt"
	CallJWTGetBool            CallKind = "jwtGetBool"
	CallJWTGetFloat           CallKind = "jwtGetFloat"
)

type ResultOrigin string

const (
	ResultOwned      ResultOrigin = "owned"
	ResultFrameOwned ResultOrigin = "frame-owned"
	ResultUnknown    ResultOrigin = "unknown"
)

type Param struct {
	Name string
	Type ast.Type
}

type Member struct {
	Package       string
	Name          string
	Kind          CallKind
	Params        []Param
	ReturnType    ast.Type
	ResultOrigin  ResultOrigin
	StatementOnly bool
	Detail        string
	SignatureText string
}

func (m Member) SourceName() string {
	return m.Package + "." + m.Name
}

func (m Member) Signature() string {
	if m.SignatureText != "" {
		return m.SignatureText
	}
	params := make([]string, 0, len(m.Params))
	for _, param := range m.Params {
		params = append(params, param.Name+" "+param.Type.String())
	}
	signature := m.SourceName() + "(" + strings.Join(params, ", ") + ")"
	if m.StatementOnly {
		return signature
	}
	return signature + " " + m.ReturnType.String()
}

type Package struct {
	Name    string
	Members []Member
}

var (
	SQLiteDBType      = &ast.PointerType{Elem: &ast.StructType{Name: "DB", Package: "sqlite", ResolvedName: "trux_sqlite_db"}}
	SQLiteStmtType    = &ast.PointerType{Elem: &ast.StructType{Name: "Stmt", Package: "sqlite", ResolvedName: "trux_sqlite_stmt"}}
	SQLiteRowMapType  = &ast.MapType{Key: ast.StringType, Value: ast.StringType}
	SQLiteRowListType = &ast.ListType{Elem: SQLiteRowMapType}

	JSONDocType    = &ast.PointerType{Elem: &ast.StructType{Name: "Doc", Package: "json", ResolvedName: "trux_json_doc"}}
	JSONValType    = &ast.PointerType{Elem: &ast.StructType{Name: "Val", Package: "json", ResolvedName: "trux_json_val"}}
	JSONMutDocType = &ast.PointerType{Elem: &ast.StructType{Name: "MutDoc", Package: "json", ResolvedName: "trux_json_mut_doc"}}
	JSONMutValType = &ast.PointerType{Elem: &ast.StructType{Name: "MutVal", Package: "json", ResolvedName: "trux_json_mut_val"}}
	JSONMapType    = &ast.MapType{Key: ast.StringType, Value: ast.StringType}

	JWTClaimsType = &ast.PointerType{Elem: &ast.StructType{Name: "Claims", Package: "jwt", ResolvedName: "trux_jwt_claims"}}
)

var packages = []Package{
	{
		Name: "io",
		Members: []Member{
			{
				Package:      "io",
				Name:         "readLine",
				Kind:         CallReadLine,
				ReturnType:   ast.StringType,
				ResultOrigin: ResultFrameOwned,
				Detail:       "read one line from stdin",
			},
			{
				Package:      "io",
				Name:         "readInt",
				Kind:         CallReadInt,
				ReturnType:   ast.IntType,
				ResultOrigin: ResultOwned,
				Detail:       "read an integer from stdin",
			},
			{
				Package:      "io",
				Name:         "readFloat",
				Kind:         CallReadFloat,
				ReturnType:   ast.FloatType,
				ResultOrigin: ResultOwned,
				Detail:       "read a float from stdin",
			},
			{
				Package:      "io",
				Name:         "readBool",
				Kind:         CallReadBool,
				ReturnType:   ast.BoolType,
				ResultOrigin: ResultOwned,
				Detail:       "read a boolean from stdin",
			},
			{
				Package: "io",
				Name:    "readFile",
				Kind:    CallReadFile,
				Params: []Param{
					{Name: "path", Type: ast.StringType},
				},
				ReturnType:   &ast.TupleType{Types: []ast.Type{ast.StringType, ast.ErrorType}},
				ResultOrigin: ResultFrameOwned,
				Detail:       "read a file into a string",
			},
			{
				Package: "io",
				Name:    "writeFile",
				Kind:    CallWriteFile,
				Params: []Param{
					{Name: "path", Type: ast.StringType},
					{Name: "contents", Type: ast.StringType},
				},
				ReturnType:    ast.ErrorType,
				ResultOrigin:  ResultFrameOwned,
				StatementOnly: false,
				Detail:        "write a string to a file",
			},
			{
				Package: "io",
				Name:    "createDir",
				Kind:    CallCreateDir,
				Params: []Param{
					{Name: "path", Type: ast.StringType},
				},
				ReturnType:   ast.ErrorType,
				ResultOrigin: ResultFrameOwned,
				Detail:       "create a directory",
			},
			{
				Package: "io",
				Name:    "remove",
				Kind:    CallRemove,
				Params: []Param{
					{Name: "path", Type: ast.StringType},
				},
				ReturnType:   ast.ErrorType,
				ResultOrigin: ResultFrameOwned,
				Detail:       "remove a file or empty directory",
			},
			{
				Package: "io",
				Name:    "readDir",
				Kind:    CallReadDir,
				Params: []Param{
					{Name: "path", Type: ast.StringType},
				},
				ReturnType:   &ast.TupleType{Types: []ast.Type{&ast.ListType{Elem: ast.StringType}, ast.ErrorType}},
				ResultOrigin: ResultFrameOwned,
				Detail:       "read directory entries sorted lexicographically",
			},
			{
				Package: "io",
				Name:    "exists",
				Kind:    CallExists,
				Params: []Param{
					{Name: "path", Type: ast.StringType},
				},
				ReturnType:   ast.BoolType,
				ResultOrigin: ResultOwned,
				Detail:       "check if a file or directory exists",
			},
			{
				Package: "io",
				Name:    "isFile",
				Kind:    CallIsFile,
				Params: []Param{
					{Name: "path", Type: ast.StringType},
				},
				ReturnType:   ast.BoolType,
				ResultOrigin: ResultOwned,
				Detail:       "check if path is a regular file",
			},
			{
				Package: "io",
				Name:    "isDir",
				Kind:    CallIsDir,
				Params: []Param{
					{Name: "path", Type: ast.StringType},
				},
				ReturnType:   ast.BoolType,
				ResultOrigin: ResultOwned,
				Detail:       "check if path is a directory",
			},
			{
				Package:      "io",
				Name:         "env",
				Kind:         CallEnv,
				ReturnType:   &ast.MapType{Key: ast.StringType, Value: ast.StringType},
				ResultOrigin: ResultFrameOwned,
				Detail:       "copy of the process environment with .env overlaid",
			},
		},
	},
	{
		Name: "csv",
		Members: []Member{
			{
				Package: "csv",
				Name:    "read",
				Kind:    CallReadCSV,
				Params: []Param{
					{Name: "path", Type: ast.StringType},
					{Name: "columns", Type: ast.IntType},
				},
				ReturnType:   &ast.TupleType{Types: []ast.Type{&ast.ListType{Elem: ast.StringType}, ast.ErrorType}},
				ResultOrigin: ResultFrameOwned,
				Detail:       "read a CSV file into row-major cells",
			},
			{
				Package: "csv",
				Name:    "write",
				Kind:    CallWriteCSV,
				Params: []Param{
					{Name: "path", Type: ast.StringType},
					{Name: "cells", Type: &ast.ListType{Elem: ast.StringType}},
					{Name: "columns", Type: ast.IntType},
				},
				ReturnType:    ast.ErrorType,
				ResultOrigin:  ResultFrameOwned,
				StatementOnly: false,
				Detail:        "write row-major cells to a CSV file",
			},
		},
	},
	{
		Name: "errors",
		Members: []Member{
			{
				Package: "errors",
				Name:    "new",
				Kind:    CallErrorsNew,
				Params: []Param{
					{Name: "msg", Type: ast.StringType},
				},
				ReturnType:   ast.ErrorType,
				ResultOrigin: ResultFrameOwned,
				Detail:       "create a new error with a message",
			},
			{
				Package: "errors",
				Name:    "message",
				Kind:    CallErrorsMessage,
				Params: []Param{
					{Name: "err", Type: ast.ErrorType},
				},
				ReturnType:   ast.StringType,
				ResultOrigin: ResultFrameOwned,
				Detail:       "get the message string of an error",
			},
		},
	},
	{
		Name: "time",
		Members: []Member{
			{Package: "time", Name: "nowUnixMillis", Kind: CallTimeNowUnixMillis, ReturnType: ast.IntType, ResultOrigin: ResultOwned, Detail: "current Unix wall-clock time in milliseconds"},
			{Package: "time", Name: "monotonicNanos", Kind: CallTimeMonotonicNanos, ReturnType: ast.IntType, ResultOrigin: ResultOwned, Detail: "monotonic timestamp in nanoseconds for elapsed-time measurement"},
			{Package: "time", Name: "sleepMillis", Kind: CallTimeSleepMillis, Params: []Param{{Name: "ms", Type: ast.IntType}}, ReturnType: ast.StringType, ResultOrigin: ResultUnknown, StatementOnly: true, Detail: "sleep for a number of milliseconds"},
		},
	},
	{
		Name: "http",
		Members: []Member{
			{Package: "http", Name: "listen", Kind: CallHTTPListen, StatementOnly: true, Detail: "listen for HTTP requests on 0.0.0.0 with auto-detected cores", SignatureText: "http.listen(port int)"},
			{Package: "http", Name: "serve", Kind: CallHTTPServe, StatementOnly: true, Detail: "serve HTTP requests with explicit host, port, and worker threads", SignatureText: "http.serve(host string, port int, workers int)"},
			{Package: "http", Name: "use", Kind: CallHTTPUse, StatementOnly: true, Detail: "register global middleware", SignatureText: "http.use(middleware)"},
			{Package: "http", Name: "useAt", Kind: CallHTTPUseAt, StatementOnly: true, Detail: "register prefix-scoped middleware", SignatureText: "http.useAt(prefix string, middleware)"},
			{Package: "http", Name: "get", Kind: CallHTTPGet, StatementOnly: true, Detail: "register GET route handler", SignatureText: "http.get(path string, handler)"},
			{Package: "http", Name: "post", Kind: CallHTTPPost, StatementOnly: true, Detail: "register POST route handler", SignatureText: "http.post(path string, handler)"},
			{Package: "http", Name: "put", Kind: CallHTTPPut, StatementOnly: true, Detail: "register PUT route handler", SignatureText: "http.put(path string, handler)"},
			{Package: "http", Name: "delete", Kind: CallHTTPDelete, StatementOnly: true, Detail: "register DELETE route handler", SignatureText: "http.delete(path string, handler)"},
			{Package: "http", Name: "patch", Kind: CallHTTPPatch, StatementOnly: true, Detail: "register PATCH route handler", SignatureText: "http.patch(path string, handler)"},
			{Package: "http", Name: "route", Kind: CallHTTPRoute, StatementOnly: true, Detail: "register custom method route handler", SignatureText: "http.route(method string, path string, handler)"},
			{Package: "http", Name: "notFound", Kind: CallHTTPNotFound, StatementOnly: true, Detail: "register custom 404 handler", SignatureText: "http.notFound(handler)"},
			{Package: "http", Name: "param", Kind: CallHTTPParam, Params: []Param{{Name: "request", Type: ast.IntType}, {Name: "name", Type: ast.StringType}}, ReturnType: ast.StringType, ResultOrigin: ResultFrameOwned, Detail: "HTTP route path parameter"},
			{Package: "http", Name: "set", Kind: CallHTTPSet, Params: []Param{{Name: "request", Type: ast.IntType}, {Name: "key", Type: ast.StringType}, {Name: "value", Type: ast.StringType}}, StatementOnly: true, Detail: "set request-scoped context value"},
			{Package: "http", Name: "context", Kind: CallHTTPContext, Params: []Param{{Name: "request", Type: ast.IntType}, {Name: "key", Type: ast.StringType}}, ReturnType: ast.StringType, ResultOrigin: ResultFrameOwned, Detail: "get request-scoped context value"},
			{Package: "http", Name: "setHeader", Kind: CallHTTPSetHeader, Params: []Param{{Name: "request", Type: ast.IntType}, {Name: "name", Type: ast.StringType}, {Name: "value", Type: ast.StringType}}, StatementOnly: true, Detail: "set custom HTTP response header"},
			{Package: "http", Name: "text", Kind: CallHTTPText, Params: []Param{{Name: "request", Type: ast.IntType}, {Name: "status", Type: ast.IntType}, {Name: "body", Type: ast.StringType}}, StatementOnly: true, Detail: "respond with text/plain body"},
			{Package: "http", Name: "json", Kind: CallHTTPJSON, Params: []Param{{Name: "request", Type: ast.IntType}, {Name: "status", Type: ast.IntType}, {Name: "body", Type: ast.StringType}}, StatementOnly: true, Detail: "respond with application/json body"},
			{Package: "http", Name: "status", Kind: CallHTTPStatus, Params: []Param{{Name: "request", Type: ast.IntType}, {Name: "status", Type: ast.IntType}}, StatementOnly: true, Detail: "respond with HTTP status code only"},
			{Package: "http", Name: "method", Kind: CallHTTPMethod, Params: []Param{{Name: "request", Type: ast.IntType}}, ReturnType: ast.StringType, ResultOrigin: ResultFrameOwned, Detail: "HTTP request method"},
			{Package: "http", Name: "path", Kind: CallHTTPPath, Params: []Param{{Name: "request", Type: ast.IntType}}, ReturnType: ast.StringType, ResultOrigin: ResultFrameOwned, Detail: "HTTP request path without query string"},
			{Package: "http", Name: "query", Kind: CallHTTPQuery, Params: []Param{{Name: "request", Type: ast.IntType}}, ReturnType: ast.StringType, ResultOrigin: ResultFrameOwned, Detail: "HTTP request raw query string without question mark"},
			{Package: "http", Name: "header", Kind: CallHTTPHeader, Params: []Param{{Name: "request", Type: ast.IntType}, {Name: "name", Type: ast.StringType}}, ReturnType: ast.StringType, ResultOrigin: ResultFrameOwned, Detail: "HTTP request header value by case-insensitive name"},
			{Package: "http", Name: "body", Kind: CallHTTPBody, Params: []Param{{Name: "request", Type: ast.IntType}}, ReturnType: ast.StringType, ResultOrigin: ResultFrameOwned, Detail: "HTTP request body"},
			{Package: "http", Name: "respond", Kind: CallHTTPRespond, Params: []Param{{Name: "request", Type: ast.IntType}, {Name: "status", Type: ast.IntType}, {Name: "contentType", Type: ast.StringType}, {Name: "body", Type: ast.StringType}}, ReturnType: ast.StringType, ResultOrigin: ResultUnknown, StatementOnly: true, Detail: "write an HTTP response"},
		},
	},
	{
		Name: "image",
		Members: []Member{
			{
				Package: "image",
				Name:    "width",
				Kind:    CallImageWidth,
				Params: []Param{
					{Name: "path", Type: ast.StringType},
				},
				ReturnType:   ast.IntType,
				ResultOrigin: ResultOwned,
				Detail:       "read the width from a P3 PPM image",
			},
			{
				Package: "image",
				Name:    "height",
				Kind:    CallImageHeight,
				Params: []Param{
					{Name: "path", Type: ast.StringType},
				},
				ReturnType:   ast.IntType,
				ResultOrigin: ResultOwned,
				Detail:       "read the height from a P3 PPM image",
			},
			{
				Package: "image",
				Name:    "readPpm",
				Kind:    CallReadPPM,
				Params: []Param{
					{Name: "path", Type: ast.StringType},
				},
				ReturnType:   &ast.TupleType{Types: []ast.Type{&ast.SliceType{Elem: ast.IntType}, ast.ErrorType}},
				ResultOrigin: ResultFrameOwned,
				Detail:       "read P3 PPM RGB pixels into a flat slice",
			},
			{
				Package: "image",
				Name:    "writePpm",
				Kind:    CallWritePPM,
				Params: []Param{
					{Name: "path", Type: ast.StringType},
					{Name: "pixels", Type: &ast.SliceType{Elem: ast.IntType}},
					{Name: "width", Type: ast.IntType},
					{Name: "height", Type: ast.IntType},
				},
				ReturnType:    ast.ErrorType,
				ResultOrigin:  ResultFrameOwned,
				StatementOnly: false,
				Detail:        "write flat RGB pixels as a P3 PPM image",
			},
		},
	},
	{
		Name: "gpu",
		Members: []Member{
			{Package: "gpu", Name: "alloc", Kind: CallGPUAlloc, Detail: "allocate a device buffer", SignatureText: "gpu.alloc(n int) gpu.Buffer[T]"},
			{Package: "gpu", Name: "copyToDevice", Kind: CallGPUCopyToDevice, Detail: "copy a host slice to a device buffer", StatementOnly: true, SignatureText: "gpu.copyToDevice(host []T, device gpu.Buffer[T])"},
			{Package: "gpu", Name: "copyToHost", Kind: CallGPUCopyToHost, Detail: "copy a device buffer to a host slice", StatementOnly: true, SignatureText: "gpu.copyToHost(device gpu.Buffer[T], host []T)"},
			{Package: "gpu", Name: "free", Kind: CallGPUFree, Detail: "free a device buffer", StatementOnly: true, SignatureText: "gpu.free(device gpu.Buffer[T])"},
			{Package: "gpu", Name: "sync", Kind: CallGPUSync, Detail: "synchronize the CUDA device", StatementOnly: true},
			{Package: "gpu", Name: "from", Kind: CallGPUFrom, Detail: "allocate a device buffer and copy a host slice into it", SignatureText: "gpu.from(host []T) gpu.Buffer[T]"},
			{Package: "gpu", Name: "mount", Kind: CallGPUMount, Detail: "load a TXM1 model once and keep it on the device", StatementOnly: true, SignatureText: "gpu.mount(path string)"},
			{Package: "gpu", Name: "workspaceFloat", Kind: CallGPUWorkspaceFloat, Detail: "declare a reused float workspace", StatementOnly: true, SignatureText: "gpu.workspaceFloat(name string, n int)"},
			{Package: "gpu", Name: "workspaceInt", Kind: CallGPUWorkspaceInt, Detail: "declare a reused int workspace", StatementOnly: true, SignatureText: "gpu.workspaceInt(name string, n int)"},
			{Package: "gpu", Name: "floats", Kind: CallGPUFloats, Detail: "return a mounted float tensor", SignatureText: "gpu.floats(name string) gpu.Buffer[float]"},
			{Package: "gpu", Name: "ints", Kind: CallGPUInts, Detail: "return a mounted int tensor", SignatureText: "gpu.ints(name string) gpu.Buffer[int]"},
			{Package: "gpu", Name: "dim", Kind: CallGPUDim, ReturnType: ast.IntType, ResultOrigin: ResultOwned, Detail: "return one dimension of a mounted tensor", SignatureText: "gpu.dim(name string, axis int) int"},
			{Package: "gpu", Name: "scalar", Kind: CallGPUScalar, ReturnType: ast.FloatType, ResultOrigin: ResultOwned, Detail: "return a one-element float tensor", SignatureText: "gpu.scalar(name string) float"},
			{Package: "gpu", Name: "hostFloats", Kind: CallGPUHostFloats, Detail: "return the host copy of a mounted float tensor", SignatureText: "gpu.hostFloats(name string) []float"},
			{Package: "gpu", Name: "hostInts", Kind: CallGPUHostInts, Detail: "return the host copy of a mounted int tensor", SignatureText: "gpu.hostInts(name string) []int"},
			{Package: "gpu", Name: "borrowFloat", Kind: CallGPUBorrowFloat, Detail: "return this worker's float workspace", SignatureText: "gpu.borrowFloat(name string) gpu.Buffer[float]"},
			{Package: "gpu", Name: "borrowInt", Kind: CallGPUBorrowInt, Detail: "return this worker's int workspace", SignatureText: "gpu.borrowInt(name string) gpu.Buffer[int]"},
			{Package: "gpu", Name: "ready", Kind: CallGPUReady, Detail: "return the pinned host mirror of a float workspace after the GPU finishes", SignatureText: "gpu.ready(name string) []float"},
			{Package: "gpu", Name: "finish", Kind: CallGPUFinish, Detail: "submit the GPU work and continue on the io_uring loop", StatementOnly: true, SignatureText: "gpu.finish(request int, continuation)"},
			{Package: "gpu", Name: "to", Kind: CallGPUTo, Detail: "copy a device buffer into a host slice", StatementOnly: true, SignatureText: "gpu.to(device gpu.Buffer[T], host []T)"},
			{Package: "gpu", Name: "launch", Kind: CallGPULaunch, Detail: "launch a GPU kernel", StatementOnly: true, SignatureText: "gpu.launch(kernel, gridX int, gridY int, gridZ int, blockX int, blockY int, blockZ int, args...)"},
			{Package: "gpu", Name: "launch1d", Kind: CallGPULaunch1D, Detail: "launch a kernel over n elements", StatementOnly: true, SignatureText: "gpu.launch1d(kernel, n int, args...)"},
			{Package: "gpu", Name: "launch2d", Kind: CallGPULaunch2D, Detail: "launch a kernel over a width by height grid", StatementOnly: true, SignatureText: "gpu.launch2d(kernel, width int, height int, args...)"},
			{Package: "gpu", Name: "globalX", Kind: CallGPUCoord, ReturnType: ast.IntType, ResultOrigin: ResultOwned, Detail: "global GPU x coordinate"},
			{Package: "gpu", Name: "globalY", Kind: CallGPUCoord, ReturnType: ast.IntType, ResultOrigin: ResultOwned, Detail: "global GPU y coordinate"},
			{Package: "gpu", Name: "globalZ", Kind: CallGPUCoord, ReturnType: ast.IntType, ResultOrigin: ResultOwned, Detail: "global GPU z coordinate"},
			{Package: "gpu", Name: "threadX", Kind: CallGPUCoord, ReturnType: ast.IntType, ResultOrigin: ResultOwned, Detail: "thread x coordinate"},
			{Package: "gpu", Name: "threadY", Kind: CallGPUCoord, ReturnType: ast.IntType, ResultOrigin: ResultOwned, Detail: "thread y coordinate"},
			{Package: "gpu", Name: "threadZ", Kind: CallGPUCoord, ReturnType: ast.IntType, ResultOrigin: ResultOwned, Detail: "thread z coordinate"},
			{Package: "gpu", Name: "blockX", Kind: CallGPUCoord, ReturnType: ast.IntType, ResultOrigin: ResultOwned, Detail: "block x coordinate"},
			{Package: "gpu", Name: "blockY", Kind: CallGPUCoord, ReturnType: ast.IntType, ResultOrigin: ResultOwned, Detail: "block y coordinate"},
			{Package: "gpu", Name: "blockZ", Kind: CallGPUCoord, ReturnType: ast.IntType, ResultOrigin: ResultOwned, Detail: "block z coordinate"},
			{Package: "gpu", Name: "blockDimX", Kind: CallGPUCoord, ReturnType: ast.IntType, ResultOrigin: ResultOwned, Detail: "block x dimension"},
			{Package: "gpu", Name: "blockDimY", Kind: CallGPUCoord, ReturnType: ast.IntType, ResultOrigin: ResultOwned, Detail: "block y dimension"},
			{Package: "gpu", Name: "blockDimZ", Kind: CallGPUCoord, ReturnType: ast.IntType, ResultOrigin: ResultOwned, Detail: "block z dimension"},
		},
	},
	{
		Name: "sqlite",
		Members: []Member{
			{
				Package: "sqlite",
				Name:    "open",
				Kind:    CallSQLiteOpen,
				Params: []Param{
					{Name: "path", Type: ast.StringType},
				},
				ReturnType:   &ast.TupleType{Types: []ast.Type{SQLiteDBType, ast.ErrorType}},
				ResultOrigin: ResultOwned,
				Detail:       "open a database connection",
			},
			{
				Package: "sqlite",
				Name:    "openThreadLocal",
				Kind:    CallSQLiteOpenThreadLocal,
				Params: []Param{
					{Name: "path", Type: ast.StringType},
				},
				ReturnType:   &ast.TupleType{Types: []ast.Type{SQLiteDBType, ast.ErrorType}},
				ResultOrigin: ResultOwned,
				Detail:       "open or retrieve a thread-local database connection",
			},
			{
				Package: "sqlite",
				Name:    "open_thread_local",
				Kind:    CallSQLiteOpenThreadLocal,
				Params: []Param{
					{Name: "path", Type: ast.StringType},
				},
				ReturnType:   &ast.TupleType{Types: []ast.Type{SQLiteDBType, ast.ErrorType}},
				ResultOrigin: ResultOwned,
				Detail:       "open or retrieve a thread-local database connection",
			},
			{
				Package: "sqlite",
				Name:    "close",
				Kind:    CallSQLiteClose,
				Params: []Param{
					{Name: "db", Type: SQLiteDBType},
				},
				ReturnType:   ast.ErrorType,
				ResultOrigin: ResultOwned,
				Detail:       "close a database connection",
			},
			{
				Package: "sqlite",
				Name:    "exec",
				Kind:    CallSQLiteExec,
				Params: []Param{
					{Name: "db", Type: SQLiteDBType},
					{Name: "sql", Type: ast.StringType},
				},
				ReturnType:   ast.ErrorType,
				ResultOrigin: ResultOwned,
				Detail:       "execute a DDL/DML statement",
			},
			{
				Package: "sqlite",
				Name:    "query",
				Kind:    CallSQLiteQuery,
				Params: []Param{
					{Name: "db", Type: SQLiteDBType},
					{Name: "sql", Type: ast.StringType},
				},
				ReturnType:   &ast.TupleType{Types: []ast.Type{SQLiteRowListType, ast.ErrorType}},
				ResultOrigin: ResultOwned,
				Detail:       "execute a query returning rows as string maps",
			},
			{
				Package: "sqlite",
				Name:    "lastInsertRowId",
				Kind:    CallSQLiteLastInsertRowID,
				Params: []Param{
					{Name: "db", Type: SQLiteDBType},
				},
				ReturnType:   ast.IntType,
				ResultOrigin: ResultOwned,
				Detail:       "return rowid of most recent successful INSERT",
			},
			{
				Package: "sqlite",
				Name:    "changes",
				Kind:    CallSQLiteChanges,
				Params: []Param{
					{Name: "db", Type: SQLiteDBType},
				},
				ReturnType:   ast.IntType,
				ResultOrigin: ResultOwned,
				Detail:       "return count of rows modified by most recent statement",
			},
			{
				Package: "sqlite",
				Name:    "totalChanges",
				Kind:    CallSQLiteTotalChanges,
				Params: []Param{
					{Name: "db", Type: SQLiteDBType},
				},
				ReturnType:   ast.IntType,
				ResultOrigin: ResultOwned,
				Detail:       "return total count of rows modified since database opened",
			},
			{
				Package: "sqlite",
				Name:    "begin",
				Kind:    CallSQLiteBegin,
				Params: []Param{
					{Name: "db", Type: SQLiteDBType},
				},
				ReturnType:   ast.ErrorType,
				ResultOrigin: ResultOwned,
				Detail:       "begin a transaction",
			},
			{
				Package: "sqlite",
				Name:    "commit",
				Kind:    CallSQLiteCommit,
				Params: []Param{
					{Name: "db", Type: SQLiteDBType},
				},
				ReturnType:   ast.ErrorType,
				ResultOrigin: ResultOwned,
				Detail:       "commit the active transaction",
			},
			{
				Package: "sqlite",
				Name:    "rollback",
				Kind:    CallSQLiteRollback,
				Params: []Param{
					{Name: "db", Type: SQLiteDBType},
				},
				ReturnType:   ast.ErrorType,
				ResultOrigin: ResultOwned,
				Detail:       "rollback the active transaction",
			},
			{
				Package: "sqlite",
				Name:    "prepare",
				Kind:    CallSQLitePrepare,
				Params: []Param{
					{Name: "db", Type: SQLiteDBType},
					{Name: "sql", Type: ast.StringType},
				},
				ReturnType:   &ast.TupleType{Types: []ast.Type{SQLiteStmtType, ast.ErrorType}},
				ResultOrigin: ResultOwned,
				Detail:       "compile SQL into a prepared statement",
			},
			{
				Package: "sqlite",
				Name:    "step",
				Kind:    CallSQLiteStep,
				Params: []Param{
					{Name: "stmt", Type: SQLiteStmtType},
				},
				ReturnType:   &ast.TupleType{Types: []ast.Type{ast.BoolType, ast.ErrorType}},
				ResultOrigin: ResultOwned,
				Detail:       "step a prepared statement to the next row",
			},
			{
				Package: "sqlite",
				Name:    "reset",
				Kind:    CallSQLiteReset,
				Params: []Param{
					{Name: "stmt", Type: SQLiteStmtType},
				},
				ReturnType:   ast.ErrorType,
				ResultOrigin: ResultOwned,
				Detail:       "reset a prepared statement back to its initial state",
			},
			{
				Package: "sqlite",
				Name:    "clearBindings",
				Kind:    CallSQLiteClearBindings,
				Params: []Param{
					{Name: "stmt", Type: SQLiteStmtType},
				},
				ReturnType:   ast.ErrorType,
				ResultOrigin: ResultOwned,
				Detail:       "clear all bound parameters to NULL",
			},
			{
				Package: "sqlite",
				Name:    "finalize",
				Kind:    CallSQLiteFinalize,
				Params: []Param{
					{Name: "stmt", Type: SQLiteStmtType},
				},
				ReturnType:   ast.ErrorType,
				ResultOrigin: ResultOwned,
				Detail:       "destroy a prepared statement object",
			},
			{
				Package: "sqlite",
				Name:    "bindInt",
				Kind:    CallSQLiteBindInt,
				Params: []Param{
					{Name: "stmt", Type: SQLiteStmtType},
					{Name: "index", Type: ast.IntType},
					{Name: "val", Type: ast.IntType},
				},
				ReturnType:   ast.ErrorType,
				ResultOrigin: ResultOwned,
				Detail:       "bind an integer to a 1-based parameter index",
			},
			{
				Package: "sqlite",
				Name:    "bindFloat",
				Kind:    CallSQLiteBindFloat,
				Params: []Param{
					{Name: "stmt", Type: SQLiteStmtType},
					{Name: "index", Type: ast.IntType},
					{Name: "val", Type: ast.FloatType},
				},
				ReturnType:   ast.ErrorType,
				ResultOrigin: ResultOwned,
				Detail:       "bind a float to a 1-based parameter index",
			},
			{
				Package: "sqlite",
				Name:    "bindText",
				Kind:    CallSQLiteBindText,
				Params: []Param{
					{Name: "stmt", Type: SQLiteStmtType},
					{Name: "index", Type: ast.IntType},
					{Name: "val", Type: ast.StringType},
				},
				ReturnType:   ast.ErrorType,
				ResultOrigin: ResultOwned,
				Detail:       "bind text to a 1-based parameter index",
			},
			{
				Package: "sqlite",
				Name:    "bindNull",
				Kind:    CallSQLiteBindNull,
				Params: []Param{
					{Name: "stmt", Type: SQLiteStmtType},
					{Name: "index", Type: ast.IntType},
				},
				ReturnType:   ast.ErrorType,
				ResultOrigin: ResultOwned,
				Detail:       "bind NULL to a 1-based parameter index",
			},
			{
				Package: "sqlite",
				Name:    "bindIndex",
				Kind:    CallSQLiteBindIndex,
				Params: []Param{
					{Name: "stmt", Type: SQLiteStmtType},
					{Name: "name", Type: ast.StringType},
				},
				ReturnType:   ast.IntType,
				ResultOrigin: ResultOwned,
				Detail:       "find parameter index by name placeholder",
			},
			{
				Package: "sqlite",
				Name:    "columnCount",
				Kind:    CallSQLiteColumnCount,
				Params: []Param{
					{Name: "stmt", Type: SQLiteStmtType},
				},
				ReturnType:   ast.IntType,
				ResultOrigin: ResultOwned,
				Detail:       "return number of columns in the result set",
			},
			{
				Package: "sqlite",
				Name:    "columnName",
				Kind:    CallSQLiteColumnName,
				Params: []Param{
					{Name: "stmt", Type: SQLiteStmtType},
					{Name: "col", Type: ast.IntType},
				},
				ReturnType:   ast.StringType,
				ResultOrigin: ResultOwned,
				Detail:       "return column name by 0-based index",
			},
			{
				Package: "sqlite",
				Name:    "columnIndex",
				Kind:    CallSQLiteColumnIndex,
				Params: []Param{
					{Name: "stmt", Type: SQLiteStmtType},
					{Name: "name", Type: ast.StringType},
				},
				ReturnType:   ast.IntType,
				ResultOrigin: ResultOwned,
				Detail:       "find 0-based column index by column name",
			},
			{
				Package: "sqlite",
				Name:    "columnType",
				Kind:    CallSQLiteColumnType,
				Params: []Param{
					{Name: "stmt", Type: SQLiteStmtType},
					{Name: "col", Type: ast.IntType},
				},
				ReturnType:   ast.StringType,
				ResultOrigin: ResultOwned,
				Detail:       "return data type of column value",
			},
			{
				Package: "sqlite",
				Name:    "columnInt",
				Kind:    CallSQLiteColumnInt,
				Params: []Param{
					{Name: "stmt", Type: SQLiteStmtType},
					{Name: "col", Type: ast.IntType},
				},
				ReturnType:   ast.IntType,
				ResultOrigin: ResultOwned,
				Detail:       "return integer value at 0-based column index",
			},
			{
				Package: "sqlite",
				Name:    "columnFloat",
				Kind:    CallSQLiteColumnFloat,
				Params: []Param{
					{Name: "stmt", Type: SQLiteStmtType},
					{Name: "col", Type: ast.IntType},
				},
				ReturnType:   ast.FloatType,
				ResultOrigin: ResultOwned,
				Detail:       "return float value at 0-based column index",
			},
			{
				Package: "sqlite",
				Name:    "columnText",
				Kind:    CallSQLiteColumnText,
				Params: []Param{
					{Name: "stmt", Type: SQLiteStmtType},
					{Name: "col", Type: ast.IntType},
				},
				ReturnType:   ast.StringType,
				ResultOrigin: ResultOwned,
				Detail:       "return text value at 0-based column index",
			},
			{
				Package: "sqlite",
				Name:    "columnIsNull",
				Kind:    CallSQLiteColumnIsNull,
				Params: []Param{
					{Name: "stmt", Type: SQLiteStmtType},
					{Name: "col", Type: ast.IntType},
				},
				ReturnType:   ast.BoolType,
				ResultOrigin: ResultOwned,
				Detail:       "check if column value is NULL at 0-based column index",
			},
		},
	},
	{
		Name: "json",
		Members: []Member{
			{Package: "json", Name: "parse", Kind: CallJSONParse, Params: []Param{{Name: "input", Type: ast.StringType}}, ReturnType: &ast.TupleType{Types: []ast.Type{JSONDocType, ast.ErrorType}}, ResultOrigin: ResultOwned, Detail: "parse a JSON string into a document"},
			{Package: "json", Name: "root", Kind: CallJSONRoot, Params: []Param{{Name: "doc", Type: JSONDocType}}, ReturnType: JSONValType, ResultOrigin: ResultFrameOwned, Detail: "get the root value of a JSON document"},
			{Package: "json", Name: "get", Kind: CallJSONGet, Params: []Param{{Name: "val", Type: JSONValType}, {Name: "key", Type: ast.StringType}}, ReturnType: JSONValType, ResultOrigin: ResultFrameOwned, Detail: "get an object member by key"},
			{Package: "json", Name: "at", Kind: CallJSONAt, Params: []Param{{Name: "val", Type: JSONValType}, {Name: "index", Type: ast.IntType}}, ReturnType: JSONValType, ResultOrigin: ResultFrameOwned, Detail: "get an array element by index"},
			{Package: "json", Name: "len", Kind: CallJSONLen, Params: []Param{{Name: "val", Type: JSONValType}}, ReturnType: ast.IntType, ResultOrigin: ResultOwned, Detail: "get the number of elements or key-value pairs"},
			{Package: "json", Name: "isObject", Kind: CallJSONIsObject, Params: []Param{{Name: "val", Type: JSONValType}}, ReturnType: ast.BoolType, ResultOrigin: ResultOwned, Detail: "check if value is a JSON object"},
			{Package: "json", Name: "isArray", Kind: CallJSONIsArray, Params: []Param{{Name: "val", Type: JSONValType}}, ReturnType: ast.BoolType, ResultOrigin: ResultOwned, Detail: "check if value is a JSON array"},
			{Package: "json", Name: "isString", Kind: CallJSONIsString, Params: []Param{{Name: "val", Type: JSONValType}}, ReturnType: ast.BoolType, ResultOrigin: ResultOwned, Detail: "check if value is a JSON string"},
			{Package: "json", Name: "isInt", Kind: CallJSONIsInt, Params: []Param{{Name: "val", Type: JSONValType}}, ReturnType: ast.BoolType, ResultOrigin: ResultOwned, Detail: "check if value is a JSON integer"},
			{Package: "json", Name: "isFloat", Kind: CallJSONIsFloat, Params: []Param{{Name: "val", Type: JSONValType}}, ReturnType: ast.BoolType, ResultOrigin: ResultOwned, Detail: "check if value is a JSON float/number"},
			{Package: "json", Name: "isBool", Kind: CallJSONIsBool, Params: []Param{{Name: "val", Type: JSONValType}}, ReturnType: ast.BoolType, ResultOrigin: ResultOwned, Detail: "check if value is a JSON boolean"},
			{Package: "json", Name: "isNull", Kind: CallJSONIsNull, Params: []Param{{Name: "val", Type: JSONValType}}, ReturnType: ast.BoolType, ResultOrigin: ResultOwned, Detail: "check if value is JSON null"},
			{Package: "json", Name: "getString", Kind: CallJSONGetString, Params: []Param{{Name: "val", Type: JSONValType}}, ReturnType: ast.StringType, ResultOrigin: ResultFrameOwned, Detail: "get value as string"},
			{Package: "json", Name: "getInt", Kind: CallJSONGetInt, Params: []Param{{Name: "val", Type: JSONValType}}, ReturnType: ast.IntType, ResultOrigin: ResultOwned, Detail: "get value as int"},
			{Package: "json", Name: "getFloat", Kind: CallJSONGetFloat, Params: []Param{{Name: "val", Type: JSONValType}}, ReturnType: ast.FloatType, ResultOrigin: ResultOwned, Detail: "get value as float"},
			{Package: "json", Name: "getBool", Kind: CallJSONGetBool, Params: []Param{{Name: "val", Type: JSONValType}}, ReturnType: ast.BoolType, ResultOrigin: ResultOwned, Detail: "get value as bool"},
			{Package: "json", Name: "fieldString", Kind: CallJSONFieldString, Params: []Param{{Name: "val", Type: JSONValType}, {Name: "key", Type: ast.StringType}}, ReturnType: ast.StringType, ResultOrigin: ResultFrameOwned, Detail: "get string field from object by key"},
			{Package: "json", Name: "fieldInt", Kind: CallJSONFieldInt, Params: []Param{{Name: "val", Type: JSONValType}, {Name: "key", Type: ast.StringType}}, ReturnType: ast.IntType, ResultOrigin: ResultOwned, Detail: "get int field from object by key"},
			{Package: "json", Name: "fieldFloat", Kind: CallJSONFieldFloat, Params: []Param{{Name: "val", Type: JSONValType}, {Name: "key", Type: ast.StringType}}, ReturnType: ast.FloatType, ResultOrigin: ResultOwned, Detail: "get float field from object by key"},
			{Package: "json", Name: "fieldBool", Kind: CallJSONFieldBool, Params: []Param{{Name: "val", Type: JSONValType}, {Name: "key", Type: ast.StringType}}, ReturnType: ast.BoolType, ResultOrigin: ResultOwned, Detail: "get bool field from object by key"},
			{Package: "json", Name: "decodeMap", Kind: CallJSONDecodeMap, Params: []Param{{Name: "input", Type: ast.StringType}}, ReturnType: &ast.TupleType{Types: []ast.Type{JSONMapType, ast.ErrorType}}, ResultOrigin: ResultOwned, Detail: "decode a JSON object string into a map[string]string"},
			{Package: "json", Name: "encodeMap", Kind: CallJSONEncodeMap, Params: []Param{{Name: "m", Type: JSONMapType}}, ReturnType: ast.StringType, ResultOrigin: ResultOwned, Detail: "encode a map[string]string into a JSON string"},
			{Package: "json", Name: "stringify", Kind: CallJSONStringify, Params: []Param{{Name: "doc", Type: JSONDocType}}, ReturnType: ast.StringType, ResultOrigin: ResultOwned, Detail: "serialize a JSON document to a string"},
			{Package: "json", Name: "valid", Kind: CallJSONValid, Params: []Param{{Name: "input", Type: ast.StringType}}, ReturnType: ast.BoolType, ResultOrigin: ResultOwned, Detail: "validate JSON syntax without creating a document"},
			{Package: "json", Name: "newDoc", Kind: CallJSONNewDoc, Params: []Param{}, ReturnType: JSONMutDocType, ResultOrigin: ResultOwned, Detail: "create a new mutable JSON document"},
			{Package: "json", Name: "newObject", Kind: CallJSONNewObject, Params: []Param{{Name: "doc", Type: JSONMutDocType}}, ReturnType: JSONMutValType, ResultOrigin: ResultFrameOwned, Detail: "create a new mutable JSON object"},
			{Package: "json", Name: "newArray", Kind: CallJSONNewArray, Params: []Param{{Name: "doc", Type: JSONMutDocType}}, ReturnType: JSONMutValType, ResultOrigin: ResultFrameOwned, Detail: "create a new mutable JSON array"},
			{Package: "json", Name: "setRoot", Kind: CallJSONSetRoot, Params: []Param{{Name: "doc", Type: JSONMutDocType}, {Name: "root", Type: JSONMutValType}}, StatementOnly: true, Detail: "set root value of mutable JSON document"},
			{Package: "json", Name: "addString", Kind: CallJSONAddString, Params: []Param{{Name: "doc", Type: JSONMutDocType}, {Name: "obj", Type: JSONMutValType}, {Name: "key", Type: ast.StringType}, {Name: "val", Type: ast.StringType}}, StatementOnly: true, Detail: "add string key-value to mutable object"},
			{Package: "json", Name: "addInt", Kind: CallJSONAddInt, Params: []Param{{Name: "doc", Type: JSONMutDocType}, {Name: "obj", Type: JSONMutValType}, {Name: "key", Type: ast.StringType}, {Name: "val", Type: ast.IntType}}, StatementOnly: true, Detail: "add int key-value to mutable object"},
			{Package: "json", Name: "addFloat", Kind: CallJSONAddFloat, Params: []Param{{Name: "doc", Type: JSONMutDocType}, {Name: "obj", Type: JSONMutValType}, {Name: "key", Type: ast.StringType}, {Name: "val", Type: ast.FloatType}}, StatementOnly: true, Detail: "add float key-value to mutable object"},
			{Package: "json", Name: "addBool", Kind: CallJSONAddBool, Params: []Param{{Name: "doc", Type: JSONMutDocType}, {Name: "obj", Type: JSONMutValType}, {Name: "key", Type: ast.StringType}, {Name: "val", Type: ast.BoolType}}, StatementOnly: true, Detail: "add bool key-value to mutable object"},
			{Package: "json", Name: "addNull", Kind: CallJSONAddNull, Params: []Param{{Name: "doc", Type: JSONMutDocType}, {Name: "obj", Type: JSONMutValType}, {Name: "key", Type: ast.StringType}}, StatementOnly: true, Detail: "add null key-value to mutable object"},
			{Package: "json", Name: "addVal", Kind: CallJSONAddVal, Params: []Param{{Name: "doc", Type: JSONMutDocType}, {Name: "obj", Type: JSONMutValType}, {Name: "key", Type: ast.StringType}, {Name: "val", Type: JSONMutValType}}, StatementOnly: true, Detail: "add value key-value to mutable object"},
			{Package: "json", Name: "appendVal", Kind: CallJSONAppendVal, Params: []Param{{Name: "arr", Type: JSONMutValType}, {Name: "val", Type: JSONMutValType}}, StatementOnly: true, Detail: "append value to mutable array"},
			{Package: "json", Name: "write", Kind: CallJSONWrite, Params: []Param{{Name: "doc", Type: JSONMutDocType}}, ReturnType: ast.StringType, ResultOrigin: ResultOwned, Detail: "serialize a mutable JSON document to a string"},
		},
	},
	{
		Name: "jwt",
		Members: []Member{
			{Package: "jwt", Name: "sign", Kind: CallJWTSign, Params: []Param{{Name: "claims", Type: ast.StringType}, {Name: "secret", Type: ast.StringType}}, ReturnType: &ast.TupleType{Types: []ast.Type{ast.StringType, ast.ErrorType}}, ResultOrigin: ResultOwned, Detail: "sign a JSON object as an HS256 JWT"},
			{Package: "jwt", Name: "verify", Kind: CallJWTVerify, Params: []Param{{Name: "token", Type: ast.StringType}, {Name: "secret", Type: ast.StringType}}, ReturnType: &ast.TupleType{Types: []ast.Type{JWTClaimsType, ast.ErrorType}}, ResultOrigin: ResultOwned, Detail: "verify an HS256 JWT and return its claims"},
			{Package: "jwt", Name: "getString", Kind: CallJWTGetString, Params: []Param{{Name: "claims", Type: JWTClaimsType}, {Name: "key", Type: ast.StringType}}, ReturnType: ast.StringType, ResultOrigin: ResultFrameOwned, Detail: "read a string claim"},
			{Package: "jwt", Name: "getInt", Kind: CallJWTGetInt, Params: []Param{{Name: "claims", Type: JWTClaimsType}, {Name: "key", Type: ast.StringType}}, ReturnType: ast.IntType, ResultOrigin: ResultOwned, Detail: "read an integer claim"},
			{Package: "jwt", Name: "getBool", Kind: CallJWTGetBool, Params: []Param{{Name: "claims", Type: JWTClaimsType}, {Name: "key", Type: ast.StringType}}, ReturnType: ast.BoolType, ResultOrigin: ResultOwned, Detail: "read a boolean claim"},
			{Package: "jwt", Name: "getFloat", Kind: CallJWTGetFloat, Params: []Param{{Name: "claims", Type: JWTClaimsType}, {Name: "key", Type: ast.StringType}}, ReturnType: ast.FloatType, ResultOrigin: ResultOwned, Detail: "read a float claim"},
		},
	},
}

var packageByName = func() map[string]Package {
	out := map[string]Package{}
	for _, pkg := range packages {
		out[pkg.Name] = pkg
	}
	return out
}()

var membersByPackage = func() map[string]map[string]Member {
	out := map[string]map[string]Member{}
	for _, pkg := range packages {
		members := map[string]Member{}
		for _, member := range pkg.Members {
			members[member.Name] = member
		}
		out[pkg.Name] = members
	}
	return out
}()

var legacyReplacements = map[string]Member{
	"readLine":  mustMember("io", "readLine"),
	"readInt":   mustMember("io", "readInt"),
	"readFloat": mustMember("io", "readFloat"),
	"readBool":  mustMember("io", "readBool"),
	"readFile":  mustMember("io", "readFile"),
	"writeFile": mustMember("io", "writeFile"),
	"readCsv":   mustMember("csv", "read"),
	"writeCsv":  mustMember("csv", "write"),
}

func Packages() []Package {
	out := append([]Package(nil), packages...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func LookupPackage(name string) (Package, bool) {
	pkg, ok := packageByName[name]
	return pkg, ok
}

func LookupMember(packageName string, memberName string) (Member, bool) {
	members, ok := membersByPackage[packageName]
	if !ok {
		return Member{}, false
	}
	member, ok := members[memberName]
	return member, ok
}

func LookupLegacy(name string) (Member, bool) {
	member, ok := legacyReplacements[name]
	return member, ok
}

func IsPackage(name string) bool {
	_, ok := packageByName[name]
	return ok
}

func mustMember(packageName string, memberName string) Member {
	member, ok := LookupMember(packageName, memberName)
	if !ok {
		panic("missing stdlib member " + packageName + "." + memberName)
	}
	return member
}
