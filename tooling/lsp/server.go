package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/claudioscheer/trux/internal/ast"
	"github.com/claudioscheer/trux/internal/formatter"
	"github.com/claudioscheer/trux/internal/language"
	modules "github.com/claudioscheer/trux/internal/modules"
	"github.com/claudioscheer/trux/internal/parser"
	"github.com/claudioscheer/trux/internal/token"
	semtypes "github.com/claudioscheer/trux/internal/types"
)

const (
	jsonrpcVersion = "2.0"

	diagnosticSeverityError = 1

	completionKindFunction = 3
	completionKindModule   = 9
	completionKindVariable = 6
	completionKindKeyword  = 14
	completionKindType     = 25
)

type Server struct {
	in        *bufio.Reader
	out       io.Writer
	documents map[string]string
	shutdown  bool
}

func NewServer(in *bufio.Reader, out io.Writer) *Server {
	return &Server{
		in:        in,
		out:       out,
		documents: map[string]string{},
	}
}

func (s *Server) Run() error {
	for {
		msg, err := readMessage(s.in)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		if msg.Method == "exit" {
			if s.shutdown {
				return nil
			}
			return fmt.Errorf("received exit before shutdown")
		}

		if msg.ID == nil {
			if err := s.handleNotification(msg); err != nil {
				return err
			}
			continue
		}

		result, err := s.handleRequest(msg)
		if err != nil {
			if writeErr := writeError(s.out, msg.ID, -32603, err.Error()); writeErr != nil {
				return writeErr
			}
			continue
		}
		if err := writeResponse(s.out, msg.ID, result); err != nil {
			return err
		}
	}
}

func (s *Server) handleRequest(msg rpcMessage) (any, error) {
	switch msg.Method {
	case "initialize":
		return initializeResult(), nil
	case "shutdown":
		s.shutdown = true
		return nil, nil
	case "textDocument/formatting":
		return s.handleFormatting(msg.Params)
	case "textDocument/definition":
		return s.handleDefinition(msg.Params)
	case "textDocument/references":
		return s.handleReferences(msg.Params)
	case "textDocument/completion":
		return s.handleCompletion(msg.Params)
	case "textDocument/hover":
		return s.handleHover(msg.Params)
	default:
		return nil, fmt.Errorf("method not found: %s", msg.Method)
	}
}

func (s *Server) handleNotification(msg rpcMessage) error {
	switch msg.Method {
	case "initialized":
		return nil
	case "textDocument/didOpen":
		var params didOpenParams
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return err
		}
		s.documents[params.TextDocument.URI] = params.TextDocument.Text
		return s.publishDiagnostics(params.TextDocument.URI)
	case "textDocument/didChange":
		var params didChangeParams
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return err
		}
		if len(params.ContentChanges) > 0 {
			s.documents[params.TextDocument.URI] = params.ContentChanges[len(params.ContentChanges)-1].Text
		}
		return s.publishDiagnostics(params.TextDocument.URI)
	case "textDocument/didClose":
		var params didCloseParams
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return err
		}
		delete(s.documents, params.TextDocument.URI)
		return s.publishDiagnosticsFor(params.TextDocument.URI, nil)
	default:
		return nil
	}
}

func (s *Server) publishDiagnostics(uri string) error {
	text := s.documents[uri]
	diagnostics := diagnosticsForDocuments(uri, text, s.documents)
	return s.publishDiagnosticsFor(uri, diagnostics)
}

func (s *Server) publishDiagnosticsFor(uri string, diagnostics []Diagnostic) error {
	if diagnostics == nil {
		diagnostics = []Diagnostic{}
	}

	return writeNotification(s.out, "textDocument/publishDiagnostics", map[string]any{
		"uri":         uri,
		"diagnostics": diagnostics,
	})
}

func (s *Server) handleFormatting(raw json.RawMessage) (any, error) {
	var params documentParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, err
	}

	text, ok := s.documents[params.TextDocument.URI]
	if !ok {
		var err error
		text, err = readURI(params.TextDocument.URI)
		if err != nil {
			return nil, err
		}
	}

	path := pathFromURI(params.TextDocument.URI)
	formatted, err := formatter.Format(path, text)
	if err != nil {
		return nil, err
	}
	if formatted == text {
		return []TextEdit{}, nil
	}

	return []TextEdit{
		{
			Range:   fullRange(text),
			NewText: formatted,
		},
	}, nil
}

func (s *Server) handleHover(raw json.RawMessage) (any, error) {
	var params hoverParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, err
	}

	if hover, ok, err := s.hoverImportPath(params.TextDocument.URI, params.Position); err != nil || ok {
		return hover, err
	}
	if hover, ok, err := s.hoverFunction(params.TextDocument.URI, params.Position); err != nil || ok {
		return hover, err
	}

	text, err := s.documentText(params.TextDocument.URI)
	if err != nil {
		return nil, nil
	}
	word, ok := wordAt(text, params.Position)
	if !ok {
		return nil, nil
	}

	value, ok := hoverText[word]
	if !ok {
		return nil, nil
	}

	return Hover{
		Contents: MarkupContent{
			Kind:  "markdown",
			Value: value,
		},
		Range: rangeForWord(text, params.Position),
	}, nil
}

func initializeResult() map[string]any {
	return map[string]any{
		"capabilities": map[string]any{
			"textDocumentSync": map[string]any{
				"openClose": true,
				"change":    1,
			},
			"documentFormattingProvider": true,
			"definitionProvider":         true,
			"referencesProvider":         true,
			"hoverProvider":              true,
			"completionProvider": map[string]any{
				"triggerCharacters": []string{".", "\"", "/"},
			},
		},
		"serverInfo": map[string]any{
			"name":    "trux-lsp",
			"version": language.Version,
		},
	}
}

func diagnosticsFor(uri string, text string) []Diagnostic {
	return diagnosticsForDocuments(uri, text, nil)
}

func diagnosticsForDocuments(uri string, text string, documents map[string]string) []Diagnostic {
	path := pathFromURI(uri)
	program, err := parser.ParseFile(path, text)
	if err != nil {
		return []Diagnostic{diagnosticFromError(text, err)}
	}

	if isTestFile(path) {
		if err := prepareTestFile(program); err != nil {
			return []Diagnostic{diagnosticFromError(text, err)}
		}
	}

	if len(program.Imports) == 0 || isTestFile(path) {
		ensureMain(program)
		if _, err := semtypes.Check(program); err != nil {
			return []Diagnostic{diagnosticFromError(text, err)}
		}
		return nil
	}

	sources := documentSources(documents)
	sources[normalizePath(path)] = text
	result, err := modules.LoadWithSources(path, sources)
	if err != nil {
		return []Diagnostic{diagnosticFromError(text, err)}
	}
	ensureMain(result.Program)
	if _, err := semtypes.Check(result.Program); err != nil {
		return []Diagnostic{diagnosticFromError(text, err)}
	}

	return nil
}

func isTestFile(path string) bool {
	return strings.HasSuffix(path, ".test.tx")
}

func prepareTestFile(program *ast.Program) error {
	for _, fn := range program.Functions {
		if !strings.HasPrefix(fn.Name, "test") {
			continue
		}
		if fn.Kernel {
			return fmt.Errorf("%s:%d:%d: test function %s must not be a kernel", fn.NamePos.File, fn.NamePos.Line, fn.NamePos.Column, fn.Name)
		}
		if len(fn.Params) != 0 {
			return fmt.Errorf("%s:%d:%d: test function %s must not have parameters", fn.NamePos.File, fn.NamePos.Line, fn.NamePos.Column, fn.Name)
		}
		if fn.ReturnType != nil {
			return fmt.Errorf("%s:%d:%d: test function %s must not declare a return type", fn.NamePos.File, fn.NamePos.Line, fn.NamePos.Column, fn.Name)
		}
		if blockHasReturn(fn.Body.Statements) {
			return fmt.Errorf("%s:%d:%d: test function %s must not return explicitly", fn.NamePos.File, fn.NamePos.Line, fn.NamePos.Column, fn.Name)
		}
		fn.ReturnType = ast.IntType
		retVal := &ast.IntLiteral{Start: fn.Body.End, Value: "0"}
		fn.Body.Statements = append(fn.Body.Statements, &ast.ReturnStmt{
			Start:  fn.Body.End,
			Value:  retVal,
			Values: []ast.Expression{retVal},
		})
	}
	return nil
}

func blockHasReturn(stmts []ast.Statement) bool {
	for _, stmt := range stmts {
		switch stmt := stmt.(type) {
		case *ast.ReturnStmt:
			return true
		case *ast.IfStmt:
			if blockHasReturn(stmt.Then.Statements) {
				return true
			}
			if stmt.Else != nil && blockHasReturn(stmt.Else.Statements) {
				return true
			}
		case *ast.ForStmt:
			if blockHasReturn(stmt.Body.Statements) {
				return true
			}
		}
	}
	return false
}

func ensureMain(program *ast.Program) {
	if hasMain(program) {
		return
	}
	pos := token.Position{Line: 1, Column: 1}
	program.Functions = append(program.Functions, &ast.FuncDecl{
		Pos:        pos,
		NamePos:    pos,
		Name:       "main",
		ReturnType: ast.IntType,
		Body: ast.Block{
			Start: pos,
			End:   pos,
			Statements: []ast.Statement{
				&ast.ReturnStmt{Start: pos, Value: &ast.IntLiteral{Start: pos, Value: "0"}, Values: []ast.Expression{&ast.IntLiteral{Start: pos, Value: "0"}}},
			},
		},
	})
}

func documentSources(documents map[string]string) map[string]string {
	sources := map[string]string{}
	for uri, text := range documents {
		sources[normalizePath(pathFromURI(uri))] = text
	}
	return sources
}

func diagnosticFromError(text string, err error) Diagnostic {
	pos := token.Position{Line: 1, Column: 1}
	message := err.Error()

	var moduleErr *modules.Error
	if errors.As(err, &moduleErr) {
		pos = moduleErr.Pos
		message = moduleErr.Msg
		if moduleErr.Source != "" {
			text = moduleErr.Source
		}
	}

	var parseErr *parser.ParseError
	if errors.As(err, &parseErr) {
		pos = parseErr.Pos
		message = parseErr.Msg
	}

	var typeErr *semtypes.Error
	if errors.As(err, &typeErr) {
		pos = typeErr.Pos
		message = typeErr.Msg
	}

	start := positionFromToken(pos)
	return Diagnostic{
		Range: Range{
			Start: start,
			End: Position{
				Line:      start.Line,
				Character: start.Character + errorSpan(text, start),
			},
		},
		Severity: diagnosticSeverityError,
		Source:   "trux",
		Message:  message,
	}
}

func hasMain(program *ast.Program) bool {
	for _, fn := range program.Functions {
		if fn.Name == "main" {
			return true
		}
	}
	return false
}

func positionFromToken(pos token.Position) Position {
	line := pos.Line - 1
	if line < 0 {
		line = 0
	}
	character := pos.Column - 1
	if character < 0 {
		character = 0
	}
	return Position{Line: line, Character: character}
}

func errorSpan(text string, pos Position) int {
	lines := strings.Split(text, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return 1
	}

	line := []rune(lines[pos.Line])
	if pos.Character < 0 || pos.Character >= len(line) {
		return 1
	}

	if !isWordRune(line[pos.Character]) {
		return 1
	}

	end := pos.Character
	for end < len(line) && isWordRune(line[end]) {
		end++
	}
	if end == pos.Character {
		return 1
	}

	return end - pos.Character
}

func fullRange(text string) Range {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return Range{}
	}

	line := len(lines) - 1
	character := len([]rune(lines[line]))
	return Range{
		Start: Position{Line: 0, Character: 0},
		End:   Position{Line: line, Character: character},
	}
}

func completionList() CompletionList {
	return CompletionList{
		IsIncomplete: false,
		Items:        baseCompletionItems(),
	}
}

func baseCompletionItems() []CompletionItem {
	return []CompletionItem{
		{Label: "package", Kind: completionKindKeyword, Detail: "package declaration"},
		{Label: "import", Kind: completionKindKeyword, Detail: "module or standard package import"},
		{Label: "pub", Kind: completionKindKeyword, Detail: "public declaration export"},
		{Label: "type", Kind: completionKindKeyword, Detail: "struct type declaration"},
		{Label: "struct", Kind: completionKindKeyword, Detail: "struct field list"},
		{Label: "kernel", Kind: completionKindKeyword, Detail: "GPU kernel declaration"},
		{Label: "func", Kind: completionKindKeyword, Detail: "function declaration"},
		{Label: "return", Kind: completionKindKeyword, Detail: "return statement"},
		{Label: "let", Kind: completionKindKeyword, Detail: "local binding"},
		{Label: "mut", Kind: completionKindKeyword, Detail: "mutable function parameter"},
		{Label: "if", Kind: completionKindKeyword, Detail: "conditional statement"},
		{Label: "else", Kind: completionKindKeyword, Detail: "conditional fallback"},
		{Label: "for", Kind: completionKindKeyword, Detail: "loop statement"},
		{Label: "break", Kind: completionKindKeyword, Detail: "exit the innermost loop"},
		{Label: "continue", Kind: completionKindKeyword, Detail: "skip to the next loop iteration"},
		{Label: "true", Kind: completionKindKeyword, Detail: "boolean literal"},
		{Label: "false", Kind: completionKindKeyword, Detail: "boolean literal"},
		{Label: "nil", Kind: completionKindKeyword, Detail: "absent error or pointer value"},
		{Label: "in", Kind: completionKindKeyword, Detail: "string containment operator"},
		{Label: "int", Kind: completionKindType, Detail: "integer type"},
		{Label: "float", Kind: completionKindType, Detail: "floating-point type"},
		{Label: "string", Kind: completionKindType, Detail: "string type"},
		{Label: "bool", Kind: completionKindType, Detail: "boolean type"},
		{Label: "error", Kind: completionKindType, Detail: "error value type"},
		{Label: "list", Kind: completionKindType, Detail: "mutable list type"},
		{Label: "map", Kind: completionKindType, Detail: "mutable map type"},
		{Label: "print", Kind: completionKindFunction, Detail: "print one or more scalar values"},
		{Label: "assert", Kind: completionKindFunction, Detail: "crash if a condition is false"},
		{Label: "len", Kind: completionKindFunction, Detail: "length of a string or collection"},
		{Label: "clone", Kind: completionKindFunction, Detail: "owned copy of a dynamic value"},
		{Label: "append", Kind: completionKindFunction, Detail: "append a value to a list"},
		{Label: "make", Kind: completionKindFunction, Detail: "create a slice or map"},
		{Label: "new", Kind: completionKindFunction, Detail: "allocate zeroed heap storage"},
		{Label: "delete", Kind: completionKindFunction, Detail: "remove a key from a map"},
		{Label: "keys", Kind: completionKindFunction, Detail: "list of map keys"},
		{Label: "values", Kind: completionKindFunction, Detail: "list of map values"},
	}
}

func wordAt(text string, pos Position) (string, bool) {
	lines := strings.Split(text, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return "", false
	}

	line := []rune(lines[pos.Line])
	if pos.Character < 0 || pos.Character > len(line) {
		return "", false
	}

	start := pos.Character
	if start == len(line) && start > 0 {
		start--
	}
	for start > 0 && isWordRune(line[start-1]) {
		start--
	}

	end := pos.Character
	for end < len(line) && isWordRune(line[end]) {
		end++
	}

	if start >= end {
		return "", false
	}
	return string(line[start:end]), true
}

func isWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func pathFromURI(uri string) string {
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme != "file" {
		return uri
	}

	path, err := url.PathUnescape(parsed.Path)
	if err != nil {
		return parsed.Path
	}
	return path
}

func readURI(uri string) (string, error) {
	path := pathFromURI(uri)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return string(data), nil
}

type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

func readMessage(reader *bufio.Reader) (rpcMessage, error) {
	contentLength := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return rpcMessage{}, err
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}

		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return rpcMessage{}, fmt.Errorf("invalid Content-Length %q: %w", value, err)
			}
			contentLength = n
		}
	}

	if contentLength <= 0 {
		return rpcMessage{}, fmt.Errorf("missing Content-Length")
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, body); err != nil {
		return rpcMessage{}, err
	}

	var msg rpcMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return rpcMessage{}, err
	}
	return msg, nil
}

func writeResponse(writer io.Writer, id json.RawMessage, result any) error {
	return writePayload(writer, map[string]any{
		"jsonrpc": jsonrpcVersion,
		"id":      id,
		"result":  result,
	})
}

func writeError(writer io.Writer, id json.RawMessage, code int, message string) error {
	return writePayload(writer, map[string]any{
		"jsonrpc": jsonrpcVersion,
		"id":      id,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func writeNotification(writer io.Writer, method string, params any) error {
	return writePayload(writer, map[string]any{
		"jsonrpc": jsonrpcVersion,
		"method":  method,
		"params":  params,
	})
}

func writePayload(writer io.Writer, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(writer, "Content-Length: %d\r\n\r\n", len(data)); err != nil {
		return err
	}
	_, err = writer.Write(data)
	return err
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type didOpenParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type didChangeParams struct {
	TextDocument   VersionedTextDocumentIdentifier `json:"textDocument"`
	ContentChanges []TextDocumentContentChange     `json:"contentChanges"`
}

type didCloseParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type TextDocumentContentChange struct {
	Text string `json:"text"`
}

type documentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type hoverParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"`
	Source   string `json:"source"`
	Message  string `json:"message"`
}

type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

type CompletionItem struct {
	Label               string     `json:"label"`
	Kind                int        `json:"kind"`
	Detail              string     `json:"detail,omitempty"`
	InsertText          string     `json:"insertText,omitempty"`
	FilterText          string     `json:"filterText,omitempty"`
	TextEdit            *TextEdit  `json:"textEdit,omitempty"`
	AdditionalTextEdits []TextEdit `json:"additionalTextEdits,omitempty"`
}

type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

var hoverText = map[string]string{
	"package":  "`package name` declares the package for the current file.",
	"import":   "`import \"path.tx\"` loads a relative module; `import \"io\"` and other standard package names load the standard library.",
	"pub":      "`pub` exports a function or struct type to files that import this module.",
	"type":     "`type Name struct { ... }` declares a struct type.",
	"struct":   "`struct { field T }` lists the fields of a type declaration.",
	"kernel":   "`kernel func name(params) { ... }` declares a GPU kernel.",
	"func":     "`func name(params) type { ... }` declares a function.",
	"return":   "`return expr, ...` exits the current function with one or more values.",
	"let":      "`let name type = expr` creates a local binding with an explicit type.",
	"mut":      "`mut name type` marks a function parameter as caller-mutable.",
	"if":       "`if condition { ... }` runs a block when a boolean condition is true.",
	"else":     "`else { ... }` runs when the previous `if` condition is false.",
	"for":      "`for condition { ... }`, `for { ... }`, or `for init; condition; post { ... }` repeats a block.",
	"break":    "`break` exits the innermost `for` loop.",
	"continue": "`continue` jumps to the next iteration of the innermost `for` loop.",
	"in":       "`needle in haystack` checks string containment and returns `bool`.",
	"nil":      "`nil` represents an absent `error` or pointer value.",
	"int":      "`int` is a signed integer value.",
	"float":    "`float` is a double-precision floating-point value.",
	"string":   "`string` is an immutable byte string.",
	"bool":     "`bool` is either `true` or `false`.",
	"error":    "`error` is a first-class error value; compare it with `nil`.",
	"list":     "`list[T]` is a mutable growable list for scalar element types.",
	"map":      "`map[K]V` is a mutable map with comparable keys.",
	"print":    "`print(value, ...)` writes one or more scalar values followed by a newline.",
	"assert":   "`assert(condition, message)` exits with a runtime error when condition is false.",
	"len":      "`len(value) int` returns the length of a string, array, slice, list, or map.",
	"clone":    "`clone(value) T` creates an owned copy of a string, array, slice, list, or map.",
	"append":   "`append(list, value)` mutates a list by adding one value.",
	"make":     "`make([]T, n) []T` creates zero-filled slice storage; `make(map[K]V)` creates a map.",
	"new":      "`new(T) *T` allocates zeroed heap storage for a value.",
	"delete":   "`delete(map, key)` removes a key from a map.",
	"keys":     "`keys(map) list[K]` returns the keys of a map.",
	"values":   "`values(map) list[V]` returns the values of a map.",
}

var builtinFunctionHoverText = map[string]string{
	"print":  "`print(value, ...)`",
	"assert": "`assert(condition bool, message string)`",
	"len":    "`len(value) int`",
	"clone":  "`clone(value) T`",
	"append": "`append(list, value)`",
	"make":   "`make([]T, n) []T`",
	"new":    "`new(T) *T`",
	"delete": "`delete(map, key)`",
	"keys":   "`keys(map) list[K]`",
	"values": "`values(map) list[V]`",
}
