package stdlib

import (
	"sort"
	"strings"

	"github.com/claudioscheer/trux/internal/ast"
)

type CallKind string

const (
	CallReadLine  CallKind = "readLine"
	CallReadInt   CallKind = "readInt"
	CallReadFloat CallKind = "readFloat"
	CallReadBool  CallKind = "readBool"
	CallReadFile  CallKind = "readFile"
	CallWriteFile CallKind = "writeFile"
	CallReadCSV   CallKind = "readCsv"
	CallWriteCSV  CallKind = "writeCsv"
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
}

func (m Member) SourceName() string {
	return m.Package + "." + m.Name
}

func (m Member) Signature() string {
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
				ReturnType:   ast.StringType,
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
				ReturnType:    ast.StringType,
				ResultOrigin:  ResultUnknown,
				StatementOnly: true,
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
				ReturnType:   &ast.ListType{Elem: ast.StringType},
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
				ReturnType:    ast.StringType,
				ResultOrigin:  ResultUnknown,
				StatementOnly: true,
				Detail:        "write row-major cells to a CSV file",
			},
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
