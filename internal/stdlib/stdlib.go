package stdlib

import (
	"sort"
	"strings"

	"github.com/claudioscheer/trux/internal/ast"
)

type CallKind string

const (
	CallReadLine           CallKind = "readLine"
	CallReadInt            CallKind = "readInt"
	CallReadFloat          CallKind = "readFloat"
	CallReadBool           CallKind = "readBool"
	CallReadFile           CallKind = "readFile"
	CallWriteFile          CallKind = "writeFile"
	CallReadCSV            CallKind = "readCsv"
	CallWriteCSV           CallKind = "writeCsv"
	CallImageWidth         CallKind = "imageWidth"
	CallImageHeight        CallKind = "imageHeight"
	CallReadPPM            CallKind = "readPpm"
	CallWritePPM           CallKind = "writePpm"
	CallTimeNowUnixMillis  CallKind = "nowUnixMillis"
	CallTimeMonotonicNanos CallKind = "monotonicNanos"
	CallTimeSleepMillis    CallKind = "sleepMillis"
	CallHTTPListen         CallKind = "listen"
	CallHTTPServe          CallKind = "serve"
	CallHTTPUse            CallKind = "use"
	CallHTTPUseAt          CallKind = "useAt"
	CallHTTPGet            CallKind = "get"
	CallHTTPPost           CallKind = "post"
	CallHTTPPut            CallKind = "put"
	CallHTTPDelete         CallKind = "delete"
	CallHTTPPatch          CallKind = "patch"
	CallHTTPRoute          CallKind = "route"
	CallHTTPNotFound       CallKind = "notFound"
	CallHTTPParam          CallKind = "param"
	CallHTTPSet            CallKind = "set"
	CallHTTPContext        CallKind = "context"
	CallHTTPSetHeader      CallKind = "setHeader"
	CallHTTPText           CallKind = "text"
	CallHTTPJSON           CallKind = "json"
	CallHTTPStatus         CallKind = "status"
	CallHTTPMethod         CallKind = "method"
	CallHTTPPath           CallKind = "path"
	CallHTTPQuery          CallKind = "query"
	CallHTTPHeader         CallKind = "header"
	CallHTTPBody           CallKind = "body"
	CallHTTPRespond        CallKind = "respond"
	CallGPUAlloc           CallKind = "gpuAlloc"
	CallGPUCopyToDevice    CallKind = "copyToDevice"
	CallGPUCopyToHost      CallKind = "copyToHost"
	CallGPUFree            CallKind = "gpuFree"
	CallGPUSync            CallKind = "gpuSync"
	CallGPULaunch          CallKind = "gpuLaunch"
	CallGPULaunch1D        CallKind = "gpuLaunch1d"
	CallGPULaunch2D        CallKind = "gpuLaunch2d"
	CallGPUFrom            CallKind = "gpuFrom"
	CallGPUTo              CallKind = "gpuTo"
	CallGPUCoord           CallKind = "gpuCoord"
	CallErrorsNew          CallKind = "errorsNew"
	CallErrorsMessage      CallKind = "errorsMessage"
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
