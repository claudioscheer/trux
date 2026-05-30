package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestDiagnosticsForParseError(t *testing.T) {
	diagnostics := diagnosticsFor("file:///tmp/main.tx", `package main
func main( int {
    return 0
}`)

	if len(diagnostics) != 1 {
		t.Fatalf("diagnostic count = %d, want 1", len(diagnostics))
	}
	if diagnostics[0].Range.Start.Line != 1 || diagnostics[0].Range.Start.Character != 11 {
		t.Fatalf("range start = %#v, want line 1 character 11", diagnostics[0].Range.Start)
	}
	if !strings.Contains(diagnostics[0].Message, `expected IDENT`) {
		t.Fatalf("message = %q, want parse error", diagnostics[0].Message)
	}
}

func TestDiagnosticsForSingleFileTypeError(t *testing.T) {
	diagnostics := diagnosticsFor("file:///tmp/main.tx", `package main
func main() int {
    print()
    return 0
}`)

	if len(diagnostics) != 1 {
		t.Fatalf("diagnostic count = %d, want 1", len(diagnostics))
	}
	if diagnostics[0].Message != "print expects at least 1 argument, got 0" {
		t.Fatalf("message = %q, want print argument error", diagnostics[0].Message)
	}
}

func TestDiagnosticsTypeChecksImportedPrograms(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.tx")
	libPath := filepath.Join(dir, "math.tx")
	mainSrc := `package main
import "math.tx"

func main() int {
    return math.add(1, 2)
}
`
	libSrc := `package math
pub func add(a int, b int) int {
    return a + b
}
`
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(libPath, []byte(libSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	mainURI := uriFromPath(normalizePath(mainPath))

	diagnostics := diagnosticsForDocuments(mainURI, mainSrc, map[string]string{mainURI: mainSrc})

	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}
}

func TestDiagnosticsForMissingImportedFunctionArguments(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.tx")
	libPath := filepath.Join(dir, "math.tx")
	mainSrc := `package main
import "math.tx"

func main() int {
    print(math.add())
    return 0
}
`
	libSrc := `package math
pub func add(a int, b int) int {
    return a + b
}
`
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(libPath, []byte(libSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	mainURI := uriFromPath(normalizePath(mainPath))

	diagnostics := diagnosticsForDocuments(mainURI, mainSrc, map[string]string{mainURI: mainSrc})

	if len(diagnostics) != 1 {
		t.Fatalf("diagnostic count = %d, want 1: %#v", len(diagnostics), diagnostics)
	}
	if diagnostics[0].Message != "math.add expects 2 arguments, got 0" {
		t.Fatalf("message = %q, want missing argument error", diagnostics[0].Message)
	}
	if diagnostics[0].Range.Start != positionOf(t, mainSrc, "math.add()") {
		t.Fatalf("range start = %#v, want add call", diagnostics[0].Range.Start)
	}
}

func TestPublishDiagnosticsClearsWithEmptyArray(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(bufio.NewReader(strings.NewReader("")), &out)
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.tx")
	libPath := filepath.Join(dir, "math.tx")
	mainSrc := `package main
import "math.tx"

func main() int {
    print(math.add(3, 4))
    return 0
}
`
	libSrc := `package math
pub func add(a int, b int) int {
    return a + b
}
`
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(libPath, []byte(libSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := uriFromPath(normalizePath(mainPath))
	server.documents[uri] = mainSrc

	if err := server.publishDiagnostics(uri); err != nil {
		t.Fatal(err)
	}

	wire := out.String()
	if !strings.Contains(wire, `"diagnostics":[]`) {
		t.Fatalf("notification = %q, want empty diagnostics array", wire)
	}
}

func TestFormattingReturnsFullDocumentEdit(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	uri := "file:///tmp/main.tx"
	server.documents[uri] = `package main
func main()int{
return 0
}`

	result, err := server.handleFormatting(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
	}))
	if err != nil {
		t.Fatal(err)
	}

	edits := result.([]TextEdit)
	if len(edits) != 1 {
		t.Fatalf("edit count = %d, want 1", len(edits))
	}
	want := "package main\nfunc main() int {\n\treturn 0\n}\n"
	if edits[0].NewText != want {
		t.Fatalf("new text = %q, want %q", edits[0].NewText, want)
	}
}

func TestInitializeAdvertisesDefinitionAndReferences(t *testing.T) {
	result := initializeResult()
	capabilities := result["capabilities"].(map[string]any)

	if capabilities["definitionProvider"] != true {
		t.Fatalf("definitionProvider = %#v, want true", capabilities["definitionProvider"])
	}
	if capabilities["referencesProvider"] != true {
		t.Fatalf("referencesProvider = %#v, want true", capabilities["referencesProvider"])
	}
}

func TestDefinitionForLocalVariable(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	uri := "file:///tmp/main.tx"
	src := `package main
func main() int {
    let total int = 1
    return total
}
`
	server.documents[uri] = src

	result, err := server.handleDefinition(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     positionOfAfter(t, src, "return ", "total"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	location := result.(Location)
	assertLocation(t, location, uri, positionOf(t, src, "total int"), "total")
}

func TestDefinitionForParameter(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	uri := "file:///tmp/main.tx"
	src := `package main
func echo(value int) int {
    return value
}
`
	server.documents[uri] = src

	result, err := server.handleDefinition(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     positionOfAfter(t, src, "return ", "value"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	location := result.(Location)
	assertLocation(t, location, uri, positionOf(t, src, "value int"), "value")
}

func TestDefinitionForImportedFunction(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.tx")
	libPath := filepath.Join(dir, "math.tx")
	shadowPath := filepath.Join(dir, "shadow.tx")
	mainSrc := `package main
import "math.tx"
import "shadow.tx"

func main() int {
    return math.add(1, 2)
}
`
	libSrc := `package math
pub func add(a int, b int) int {
    return a + b
}
`
	shadowSrc := `package shadow
func add(a int, b int) int {
    return a - b
}

pub func use() int {
    return add(3, 4)
}
`
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(libPath, []byte(libSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shadowPath, []byte(shadowSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	mainURI := uriFromPath(normalizePath(mainPath))
	libURI := uriFromPath(normalizePath(libPath))
	server.documents[mainURI] = mainSrc

	result, err := server.handleDefinition(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": mainURI},
		"position":     positionOfAfter(t, mainSrc, "math.", "add"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	location := result.(Location)
	assertLocation(t, location, libURI, positionOf(t, libSrc, "add(a"), "add")
}

func TestDefinitionForImportPath(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.tx")
	libPath := filepath.Join(dir, "math.tx")
	mainSrc := `package main
import "math.tx"

func main() int {
    return
}
`
	libSrc := `package math
pub func add(a int, b int) int {
    return a + b
}
`
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(libPath, []byte(libSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	mainURI := uriFromPath(normalizePath(mainPath))
	libURI := uriFromPath(normalizePath(libPath))
	server.documents[mainURI] = mainSrc

	result, err := server.handleDefinition(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": mainURI},
		"position":     positionOfAfter(t, mainSrc, "math.", "tx"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	links := result.([]LocationLink)
	if len(links) != 1 {
		t.Fatalf("link count = %d, want 1: %#v", len(links), links)
	}
	targetRange := Range{
		Start: Position{Line: 0, Character: 0},
		End:   Position{Line: 0, Character: 0},
	}
	link := links[0]
	assertRange(t, link.OriginSelectionRange, positionOf(t, mainSrc, "math.tx"), "math.tx")
	if link.TargetURI != libURI {
		t.Fatalf("target URI = %q, want %q", link.TargetURI, libURI)
	}
	if link.TargetRange != targetRange {
		t.Fatalf("target range = %#v, want %#v", link.TargetRange, targetRange)
	}
	if link.TargetSelectionRange != targetRange {
		t.Fatalf("target selection range = %#v, want %#v", link.TargetSelectionRange, targetRange)
	}
}

func TestReferencesForLocalVariable(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	uri := "file:///tmp/main.tx"
	src := `package main
func main() int {
    let total int = 1
    total = total + 1
    return total
}
`
	server.documents[uri] = src

	result, err := server.handleReferences(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     positionOfAfter(t, src, "return ", "total"),
		"context":      map[string]any{"includeDeclaration": true},
	}))
	if err != nil {
		t.Fatal(err)
	}

	locations := result.([]Location)
	if len(locations) != 4 {
		t.Fatalf("reference count = %d, want 4: %#v", len(locations), locations)
	}
	assertLocation(t, locations[0], uri, positionOf(t, src, "total int"), "total")
	assertLocation(t, locations[1], uri, positionOfAfter(t, src, "    ", "total ="), "total")
	assertLocation(t, locations[2], uri, positionOfAfter(t, src, "total = ", "total"), "total")
	assertLocation(t, locations[3], uri, positionOfAfter(t, src, "return ", "total"), "total")
}

func TestReferencesForImportedFunction(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.tx")
	libPath := filepath.Join(dir, "math.tx")
	shadowPath := filepath.Join(dir, "shadow.tx")
	mainSrc := `package main
import "math.tx"
import "shadow.tx"

func main() int {
    return math.add(1, 2)
}
`
	libSrc := `package math
pub func add(a int, b int) int {
    return a + b
}
`
	shadowSrc := `package shadow
func add(a int, b int) int {
    return a - b
}

pub func use() int {
    return add(3, 4)
}
`
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(libPath, []byte(libSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shadowPath, []byte(shadowSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	mainURI := uriFromPath(normalizePath(mainPath))
	libURI := uriFromPath(normalizePath(libPath))
	server.documents[mainURI] = mainSrc

	result, err := server.handleReferences(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": mainURI},
		"position":     positionOfAfter(t, mainSrc, "math.", "add"),
		"context":      map[string]any{"includeDeclaration": true},
	}))
	if err != nil {
		t.Fatal(err)
	}

	locations := result.([]Location)
	if len(locations) != 2 {
		t.Fatalf("reference count = %d, want 2: %#v", len(locations), locations)
	}
	assertLocation(t, locations[0], libURI, positionOf(t, libSrc, "add(a"), "add")
	assertLocation(t, locations[1], mainURI, positionOfAfter(t, mainSrc, "math.", "add"), "add")
}

func TestCompletionIncludesVisibleSymbols(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.tx")
	libPath := filepath.Join(dir, "math.tx")
	mainSrc := `package main
import "math.tx"

func helper() int {
    return 1
}

func main() int {
    let total int = helper()
    return total
}
`
	libSrc := `package math
pub func add(a int, b int) int {
    return a + b
}
`
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(libPath, []byte(libSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	mainURI := uriFromPath(normalizePath(mainPath))
	server.documents[mainURI] = mainSrc

	result, err := server.handleCompletion(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": mainURI},
		"position":     positionOfAfter(t, mainSrc, "    return total", ""),
	}))
	if err != nil {
		t.Fatal(err)
	}

	items := result.(CompletionList).Items
	want := map[string]int{
		"return": completionKindKeyword,
		"print":  completionKindFunction,
		"total":  completionKindVariable,
		"helper": completionKindFunction,
		"add":    completionKindFunction,
	}
	for label, kind := range want {
		item, ok := findCompletionItem(items, label)
		if !ok {
			t.Fatalf("missing completion %q in %#v", label, items)
		}
		if item.Kind != kind {
			t.Fatalf("completion %q kind = %d, want %d", label, item.Kind, kind)
		}
	}
}

func TestCompletionIncludesImportedFunctionsWithIncompleteBuffer(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.tx")
	libPath := filepath.Join(dir, "math.tx")
	mainSrc := `package main
import "math.tx"

func main() int {
    return
}
`
	libSrc := `package math
pub func add(a int, b int) int {
    return a + b
}
`
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(libPath, []byte(libSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	mainURI := uriFromPath(normalizePath(mainPath))
	server.documents[mainURI] = mainSrc

	result, err := server.handleCompletion(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": mainURI},
		"position":     positionOfAfter(t, mainSrc, "    return", ""),
	}))
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := findCompletionItem(result.(CompletionList).Items, "add"); !ok {
		t.Fatalf("missing imported completion add in %#v", result)
	}
}

func TestHoverReturnsKnownSymbolDocumentation(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	uri := "file:///tmp/main.tx"
	server.documents[uri] = "package main\nfunc main() int {\n\tprint(1)\n\treturn 0\n}\n"

	result, err := server.handleHover(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 2, "character": 2},
	}))
	if err != nil {
		t.Fatal(err)
	}

	hover := result.(Hover)
	if !strings.Contains(hover.Contents.Value, "print") {
		t.Fatalf("hover = %#v, want print documentation", hover)
	}
}

func TestHoverReturnsImportedFunctionSignature(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.tx")
	libPath := filepath.Join(dir, "math.tx")
	mainSrc := `package main
import "math.tx"

func main() int {
    return math.add(1, 2)
}
`
	libSrc := `package math
pub func add(a int, b int) int {
    return a + b
}
`
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(libPath, []byte(libSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	mainURI := uriFromPath(normalizePath(mainPath))
	server.documents[mainURI] = mainSrc

	result, err := server.handleHover(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": mainURI},
		"position":     positionOfAfter(t, mainSrc, "math.", "add"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	hover := result.(Hover)
	if hover.Contents.Value != "`pub func add(a int, b int) int`" {
		t.Fatalf("hover = %#v, want imported function signature", hover)
	}
	assertRange(t, hover.Range, positionOfAfter(t, mainSrc, "math.", "add"), "add")
}

func TestHoverHighlightsFullImportPath(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	uri := "file:///tmp/src/main.tx"
	src := `package main
import "../math.tx"

func main() int {
    return 0
}
`
	server.documents[uri] = src

	result, err := server.handleHover(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     positionOf(t, src, "math.tx"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	hover := result.(Hover)
	if hover.Contents.Value != "`import \"../math.tx\"`" {
		t.Fatalf("hover = %#v, want import path hover", hover)
	}
	assertRange(t, hover.Range, positionOf(t, src, "../math.tx"), "../math.tx")
}

func TestReadAndWriteMessage(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	wire := "Content-Length: " + strconv.Itoa(len(input)) + "\r\n\r\n" + input

	msg, err := readMessage(bufio.NewReader(strings.NewReader(wire)))
	if err != nil {
		t.Fatal(err)
	}
	if msg.Method != "initialize" || string(msg.ID) != "1" {
		t.Fatalf("message = %#v, want initialize request", msg)
	}

	var out bytes.Buffer
	if err := writeResponse(&out, msg.ID, map[string]string{"ok": "true"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Content-Length: ") || !strings.Contains(out.String(), `"result":{"ok":"true"}`) {
		t.Fatalf("wire response = %q", out.String())
	}
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func positionOf(t *testing.T, text string, needle string) Position {
	t.Helper()

	index := strings.Index(text, needle)
	if index < 0 {
		t.Fatalf("could not find %q in source", needle)
	}
	return positionFromOffset(text, index)
}

func positionOfAfter(t *testing.T, text string, after string, needle string) Position {
	t.Helper()

	afterIndex := strings.Index(text, after)
	if afterIndex < 0 {
		t.Fatalf("could not find prefix %q in source", after)
	}
	index := strings.Index(text[afterIndex+len(after):], needle)
	if index < 0 {
		t.Fatalf("could not find %q after %q in source", needle, after)
	}
	return positionFromOffset(text, afterIndex+len(after)+index)
}

func positionFromOffset(text string, offset int) Position {
	line := 0
	character := 0
	for i, r := range text {
		if i >= offset {
			break
		}
		if r == '\n' {
			line++
			character = 0
			continue
		}
		character++
	}
	return Position{Line: line, Character: character}
}

func assertLocation(t *testing.T, got Location, uri string, start Position, name string) {
	t.Helper()

	want := Location{
		URI: uri,
		Range: Range{
			Start: start,
			End: Position{
				Line:      start.Line,
				Character: start.Character + len([]rune(name)),
			},
		},
	}
	if got != want {
		t.Fatalf("location = %#v, want %#v", got, want)
	}
}

func assertRange(t *testing.T, got *Range, start Position, text string) {
	t.Helper()

	if got == nil {
		t.Fatalf("range = nil, want range for %q", text)
	}
	want := Range{
		Start: start,
		End: Position{
			Line:      start.Line,
			Character: start.Character + len([]rune(text)),
		},
	}
	if *got != want {
		t.Fatalf("range = %#v, want %#v", *got, want)
	}
}

func findCompletionItem(items []CompletionItem, label string) (CompletionItem, bool) {
	for _, item := range items {
		if item.Label == label {
			return item, true
		}
	}
	return CompletionItem{}, false
}
