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

	"github.com/claudioscheer/trux/internal/language"
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

func TestDiagnosticsTypeChecksSamePackageSiblings(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.tx")
	betaPath := filepath.Join(dir, "beta.tx")
	beta2Path := filepath.Join(dir, "beta2.tx")
	mainSrc := `package main
import "beta.tx"

func main() int {
    return beta.beta()
}
`
	betaSrc := `package beta
pub func beta() int {
    return value2()
}
`
	beta2Src := `package beta
func value2() int {
    return 2
}
`
	for path, src := range map[string]string{
		mainPath:  mainSrc,
		betaPath:  betaSrc,
		beta2Path: beta2Src,
	} {
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mainURI := uriFromPath(normalizePath(mainPath))
	betaURI := uriFromPath(normalizePath(betaPath))
	beta2URI := uriFromPath(normalizePath(beta2Path))

	diagnostics := diagnosticsForDocuments(mainURI, mainSrc, map[string]string{
		mainURI:  mainSrc,
		betaURI:  betaSrc,
		beta2URI: beta2Src,
	})

	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}
}

func TestDiagnosticsTypeChecksStandardLibraryImports(t *testing.T) {
	uri := "file:///tmp/main.tx"
	src := `package main
import "io"
import "csv"

func main() int {
    let line string = io.readLine()
    let cells list[string] = csv.read("input.csv", 2)
    io.writeFile("out.txt", line)
    csv.write("out.csv", cells, 2)
    return 0
}
`

	diagnostics := diagnosticsForDocuments(uri, src, map[string]string{uri: src})

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
	want := "package main\n\nfunc main() int {\n  return 0\n}\n"
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
	completionProvider := capabilities["completionProvider"].(map[string]any)
	triggerCharacters := completionProvider["triggerCharacters"].([]string)
	wantTriggers := []string{".", "\"", "/"}
	if strings.Join(triggerCharacters, ",") != strings.Join(wantTriggers, ",") {
		t.Fatalf("trigger characters = %#v, want %#v", triggerCharacters, wantTriggers)
	}
	serverInfo := result["serverInfo"].(map[string]any)
	if serverInfo["version"] != language.Version {
		t.Fatalf("server version = %#v, want language version %q", serverInfo["version"], language.Version)
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

func TestDefinitionForSamePackageSiblingPrivateFunction(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	dir := t.TempDir()
	betaPath := filepath.Join(dir, "beta.tx")
	beta2Path := filepath.Join(dir, "beta2.tx")
	betaSrc := `package beta
pub func beta() int {
    return value2()
}
`
	beta2Src := `package beta
func value2() int {
    return 2
}
`
	if err := os.WriteFile(betaPath, []byte(betaSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(beta2Path, []byte(beta2Src), 0o644); err != nil {
		t.Fatal(err)
	}
	betaURI := uriFromPath(normalizePath(betaPath))
	beta2URI := uriFromPath(normalizePath(beta2Path))
	server.documents[betaURI] = betaSrc
	server.documents[beta2URI] = beta2Src

	result, err := server.handleDefinition(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": betaURI},
		"position":     positionOfAfter(t, betaSrc, "return ", "value2"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	location := result.(Location)
	assertLocation(t, location, beta2URI, positionOf(t, beta2Src, "value2()"), "value2")
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

func TestDefinitionForImportPackageQualifier(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.tx")
	libPath := filepath.Join(dir, "math.tx")
	mainSrc := `package main
import "math.tx"

func main() int {
    return print(math.add(3, 4))
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

	result, err := server.handleDefinition(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": mainURI},
		"position":     positionOfAfter(t, mainSrc, "print(", "math"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	links, ok := result.([]LocationLink)
	if !ok {
		t.Fatalf("result = %T, want []LocationLink", result)
	}
	if len(links) != 1 {
		t.Fatalf("link count = %d, want 1: %#v", len(links), links)
	}
	link := links[0]
	assertRange(t, link.OriginSelectionRange, positionOfAfter(t, mainSrc, "print(", "math"), "math")
	if link.TargetURI != mainURI {
		t.Fatalf("target URI = %q, want %q", link.TargetURI, mainURI)
	}
	assertRange(t, &link.TargetRange, positionOf(t, mainSrc, "math.tx"), "math.tx")
	assertRange(t, &link.TargetSelectionRange, positionOf(t, mainSrc, "math.tx"), "math.tx")
}

func TestDefinitionForImportPackageQualifierUsesDeclaredPackageName(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.tx")
	libPath := filepath.Join(dir, "calc.tx")
	mainSrc := `package main
import "calc.tx"

func main() int {
    return math.add(3, 4)
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

	result, err := server.handleDefinition(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": mainURI},
		"position":     positionOfAfter(t, mainSrc, "return ", "math"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	links, ok := result.([]LocationLink)
	if !ok {
		t.Fatalf("result = %T, want []LocationLink", result)
	}
	if len(links) != 1 {
		t.Fatalf("link count = %d, want 1: %#v", len(links), links)
	}
	link := links[0]
	assertRange(t, link.OriginSelectionRange, positionOfAfter(t, mainSrc, "return ", "math"), "math")
	if link.TargetURI != mainURI {
		t.Fatalf("target URI = %q, want %q", link.TargetURI, mainURI)
	}
	assertRange(t, &link.TargetRange, positionOf(t, mainSrc, "calc.tx"), "calc.tx")
	assertRange(t, &link.TargetSelectionRange, positionOf(t, mainSrc, "calc.tx"), "calc.tx")
}

func TestDefinitionForStandardLibraryPackageQualifier(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	uri := "file:///tmp/main.tx"
	src := `package main
import "io"

func main() int {
    let line string = io.readLine()
    return len(line)
}
`
	server.documents[uri] = src

	result, err := server.handleDefinition(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     positionOfAfter(t, src, "line string = ", "io"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	links, ok := result.([]LocationLink)
	if !ok {
		t.Fatalf("result = %T, want []LocationLink", result)
	}
	if len(links) != 1 {
		t.Fatalf("link count = %d, want 1: %#v", len(links), links)
	}
	link := links[0]
	assertRange(t, link.OriginSelectionRange, positionOfAfter(t, src, "line string = ", "io"), "io")
	if link.TargetURI != uri {
		t.Fatalf("target URI = %q, want %q", link.TargetURI, uri)
	}
	assertRange(t, &link.TargetRange, positionOf(t, src, "io\""), "io")
	assertRange(t, &link.TargetSelectionRange, positionOf(t, src, "io\""), "io")
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

func TestReferencesForSamePackageSiblingPrivateFunction(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	dir := t.TempDir()
	betaPath := filepath.Join(dir, "beta.tx")
	beta2Path := filepath.Join(dir, "beta2.tx")
	betaSrc := `package beta
pub func beta() int {
    return value2()
}
`
	beta2Src := `package beta
func value2() int {
    return 2
}
`
	if err := os.WriteFile(betaPath, []byte(betaSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(beta2Path, []byte(beta2Src), 0o644); err != nil {
		t.Fatal(err)
	}
	betaURI := uriFromPath(normalizePath(betaPath))
	beta2URI := uriFromPath(normalizePath(beta2Path))
	server.documents[betaURI] = betaSrc
	server.documents[beta2URI] = beta2Src

	result, err := server.handleReferences(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": beta2URI},
		"position":     positionOf(t, beta2Src, "value2()"),
		"context":      map[string]any{"includeDeclaration": true},
	}))
	if err != nil {
		t.Fatal(err)
	}

	locations := result.([]Location)
	if len(locations) != 2 {
		t.Fatalf("reference count = %d, want 2: %#v", len(locations), locations)
	}
	assertLocation(t, locations[0], beta2URI, positionOf(t, beta2Src, "value2()"), "value2")
	assertLocation(t, locations[1], betaURI, positionOfAfter(t, betaSrc, "return ", "value2"), "value2")
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
		"return":   completionKindKeyword,
		"for":      completionKindKeyword,
		"print":    completionKindFunction,
		"total":    completionKindVariable,
		"helper":   completionKindFunction,
		"math":     completionKindModule,
		"math.add": completionKindFunction,
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
	if item, ok := findCompletionItem(items, "math.add"); !ok {
		t.Fatalf("missing completion math.add in %#v", items)
	} else {
		if item.InsertText != "math.add" {
			t.Fatalf("math.add insertText = %q, want math.add", item.InsertText)
		}
		if item.FilterText != "math.add add" {
			t.Fatalf("math.add filterText = %q, want math.add add", item.FilterText)
		}
	}
	assertMissingCompletion(t, items, "add")
	assertMissingCompletion(t, items, "while")
}

func TestCompletionIncludesForInitLocalOnlyInsideLoop(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	uri := "file:///tmp/main.tx"
	src := `package main
func main() int {
    let sum int = 0
    for let i int = 0; i < 3; i = i + 1 {
        print(i)
    }
    return sum
}
`
	server.documents[uri] = src

	insideResult, err := server.handleCompletion(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     positionOfAfter(t, src, "        print(", ""),
	}))
	if err != nil {
		t.Fatal(err)
	}
	insideItems := insideResult.(CompletionList).Items
	if _, ok := findCompletionItem(insideItems, "i"); !ok {
		t.Fatalf("missing for init local i inside loop in %#v", insideItems)
	}
	if _, ok := findCompletionItem(insideItems, "sum"); !ok {
		t.Fatalf("missing outer local sum inside loop in %#v", insideItems)
	}

	afterResult, err := server.handleCompletion(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     positionOfAfter(t, src, "    return ", "sum"),
	}))
	if err != nil {
		t.Fatal(err)
	}
	afterItems := afterResult.(CompletionList).Items
	assertMissingCompletion(t, afterItems, "i")
	if _, ok := findCompletionItem(afterItems, "sum"); !ok {
		t.Fatalf("missing outer local sum after loop in %#v", afterItems)
	}
}

func TestCompletionIncludesSamePackagePrivateFunctions(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	dir := t.TempDir()
	betaPath := filepath.Join(dir, "beta.tx")
	beta2Path := filepath.Join(dir, "beta2.tx")
	betaSrc := `package beta
pub func beta() int {
    return val
}
`
	beta2Src := `package beta
func value2() int {
    return 2
}
`
	if err := os.WriteFile(betaPath, []byte(betaSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(beta2Path, []byte(beta2Src), 0o644); err != nil {
		t.Fatal(err)
	}
	betaURI := uriFromPath(normalizePath(betaPath))
	server.documents[betaURI] = betaSrc

	result, err := server.handleCompletion(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": betaURI},
		"position":     positionOfAfter(t, betaSrc, "return ", "val"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	items := result.(CompletionList).Items
	if item, ok := findCompletionItem(items, "value2"); !ok {
		t.Fatalf("missing same-package private completion value2 in %#v", items)
	} else if item.Detail != "function from package beta" {
		t.Fatalf("value2 detail = %q, want same-package detail", item.Detail)
	}
	assertMissingCompletion(t, items, "beta.value2")
}

func TestCompletionIncludesQualifiedImportedFunctionsWithIncompleteBuffer(t *testing.T) {
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

	items := result.(CompletionList).Items
	if item, ok := findCompletionItem(items, "math"); !ok {
		t.Fatalf("missing imported package math in %#v", result)
	} else if item.Kind != completionKindModule {
		t.Fatalf("math kind = %d, want module", item.Kind)
	}
	if item, ok := findCompletionItem(items, "math.add"); !ok {
		t.Fatalf("missing imported completion math.add in %#v", result)
	} else if item.InsertText != "math.add" {
		t.Fatalf("math.add insertText = %q, want math.add", item.InsertText)
	}
	assertMissingCompletion(t, items, "add")
}

func TestCompletionIncludesStandardLibraryPackagesAndFunctions(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	uri := "file:///tmp/main.tx"
	src := `package main
import "io"
import "csv"

func main() int {
    return
}
`
	server.documents[uri] = src

	result, err := server.handleCompletion(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     positionOfAfter(t, src, "    return", ""),
	}))
	if err != nil {
		t.Fatal(err)
	}

	items := result.(CompletionList).Items
	want := map[string]int{
		"io":          completionKindModule,
		"io.readLine": completionKindFunction,
		"csv":         completionKindModule,
		"csv.read":    completionKindFunction,
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
	if item, ok := findCompletionItem(items, "io.readLine"); !ok {
		t.Fatalf("missing completion io.readLine in %#v", items)
	} else if item.FilterText != "io.readLine readLine" {
		t.Fatalf("io.readLine filterText = %q, want io.readLine readLine", item.FilterText)
	}
}

func TestCompletionIncludesMissingStandardLibraryFunctionsWithImportEdits(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	uri := "file:///tmp/main.tx"
	src := `package main

func main() int {
    return
}
`
	server.documents[uri] = src

	result, err := server.handleCompletion(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     positionOfAfter(t, src, "    return", ""),
	}))
	if err != nil {
		t.Fatal(err)
	}

	items := result.(CompletionList).Items
	ioItem, ok := findCompletionItem(items, "io.readLine")
	if !ok {
		t.Fatalf("missing completion io.readLine in %#v", items)
	}
	assertCompletionImportEdit(t, ioItem, 1, `import "io"`+"\n")
	if ioItem.InsertText != "io.readLine" {
		t.Fatalf("io.readLine insertText = %q, want io.readLine", ioItem.InsertText)
	}
	if ioItem.FilterText != "io.readLine readLine" {
		t.Fatalf("io.readLine filterText = %q, want io.readLine readLine", ioItem.FilterText)
	}

	csvItem, ok := findCompletionItem(items, "csv.read")
	if !ok {
		t.Fatalf("missing completion csv.read in %#v", items)
	}
	assertCompletionImportEdit(t, csvItem, 1, `import "csv"`+"\n")
	assertMissingCompletion(t, items, "readLine")
}

func TestCompletionDoesNotDuplicateExistingStandardLibraryImportEdit(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	uri := "file:///tmp/main.tx"
	src := `package main
import "io"

func main() int {
    return
}
`
	server.documents[uri] = src

	result, err := server.handleCompletion(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     positionOfAfter(t, src, "    return", ""),
	}))
	if err != nil {
		t.Fatal(err)
	}

	items := result.(CompletionList).Items
	ioItem, ok := findCompletionItem(items, "io.readLine")
	if !ok {
		t.Fatalf("missing completion io.readLine in %#v", items)
	}
	if len(ioItem.AdditionalTextEdits) != 0 {
		t.Fatalf("io.readLine additional edits = %#v, want none", ioItem.AdditionalTextEdits)
	}

	csvItem, ok := findCompletionItem(items, "csv.read")
	if !ok {
		t.Fatalf("missing completion csv.read in %#v", items)
	}
	assertCompletionImportEdit(t, csvItem, 2, `import "csv"`+"\n")
}

func TestCompletionIncludesStandardLibraryImportPaths(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	uri := "file:///tmp/main.tx"
	src := `package main
import "i"

func main() int {
    return 0
}
`
	server.documents[uri] = src

	result, err := server.handleCompletion(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     positionOfAfter(t, src, `import "`, "i"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	items := result.(CompletionList).Items
	if item, ok := findCompletionItem(items, "io"); !ok {
		t.Fatalf("missing standard package io in %#v", items)
	} else if item.Detail != "standard package" {
		t.Fatalf("io detail = %q, want standard package", item.Detail)
	}
	if _, ok := findCompletionItem(items, "csv"); !ok {
		t.Fatalf("missing standard package csv in %#v", items)
	}
	assertMissingCompletion(t, items, "print")
}

func TestCompletionIncludesSourceImportPathsAndDirectories(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.tx")
	utilPath := filepath.Join(dir, "util.tx")
	nestedDir := filepath.Join(dir, "math")
	emptyDir := filepath.Join(dir, "empty")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(utilPath, []byte("package util\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "add.tx"), []byte("package math\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := `package main
import "

func main() int {
    return 0
}
`
	mainURI := uriFromPath(normalizePath(mainPath))
	server.documents[mainURI] = src

	result, err := server.handleCompletion(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": mainURI},
		"position":     positionOfAfter(t, src, `import "`, ""),
	}))
	if err != nil {
		t.Fatal(err)
	}

	items := result.(CompletionList).Items
	if _, ok := findCompletionItem(items, "io"); !ok {
		t.Fatalf("missing standard package io in %#v", items)
	}
	if _, ok := findCompletionItem(items, "csv"); !ok {
		t.Fatalf("missing standard package csv in %#v", items)
	}
	if item, ok := findCompletionItem(items, "util.tx"); !ok {
		t.Fatalf("missing source file util.tx in %#v", items)
	} else {
		assertCompletionTextEdit(t, item, "util.tx")
	}
	if item, ok := findCompletionItem(items, "math/"); !ok {
		t.Fatalf("missing directory math/ in %#v", items)
	} else {
		assertCompletionTextEdit(t, item, "math/")
	}
	assertMissingCompletion(t, items, "empty/")
	assertMissingCompletion(t, items, "print")
	assertMissingCompletion(t, items, "io.readLine")
}

func TestCompletionIncludesParentSourceImportPaths(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	dir := t.TempDir()
	childDir := filepath.Join(dir, "app")
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(childDir, "main.tx")
	if err := os.WriteFile(filepath.Join(dir, "math.tx"), []byte("package math\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := `package main
import "../"

func main() int {
    return 0
}
`
	mainURI := uriFromPath(normalizePath(mainPath))
	server.documents[mainURI] = src

	result, err := server.handleCompletion(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": mainURI},
		"position":     positionOfAfter(t, src, `import "../`, ""),
	}))
	if err != nil {
		t.Fatal(err)
	}

	items := result.(CompletionList).Items
	item, ok := findCompletionItem(items, "math.tx")
	if !ok {
		t.Fatalf("missing parent source file math.tx in %#v", items)
	}
	assertCompletionTextEdit(t, item, "../math.tx")
	assertMissingCompletion(t, items, "io")
	assertMissingCompletion(t, items, "print")
}

func TestCompletionOmitsFunctionNoiseInDeclarationArea(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	uri := "file:///tmp/main.tx"
	src := `package main
io.

func main() int {
    return 0
}
`
	server.documents[uri] = src

	result, err := server.handleCompletion(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     positionOfAfter(t, src, "io.", ""),
	}))
	if err != nil {
		t.Fatal(err)
	}

	items := result.(CompletionList).Items
	if _, ok := findCompletionItem(items, "import"); !ok {
		t.Fatalf("missing declaration completion import in %#v", items)
	}
	assertMissingCompletion(t, items, "print")
	assertMissingCompletion(t, items, "readLine")
	assertMissingCompletion(t, items, "io.readLine")
}

func TestCompletionFiltersPackageMembersAfterDot(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.tx")
	mathPath := filepath.Join(dir, "math.tx")
	stringsPath := filepath.Join(dir, "strings.tx")
	mainSrc := `package main
import "math.tx"
import "strings.tx"

func helper() int {
    return 1
}

func main() int {
    let total int = 1
    return math.ad
}
`
	mathSrc := `package math
pub func add(a int, b int) int {
    return a + b
}

func hidden() int {
    return 0
}
`
	stringsSrc := `package strings
pub func trim(value string) string {
    return value
}
`
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mathPath, []byte(mathSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stringsPath, []byte(stringsSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	mainURI := uriFromPath(normalizePath(mainPath))
	server.documents[mainURI] = mainSrc

	result, err := server.handleCompletion(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": mainURI},
		"position":     positionOfAfter(t, mainSrc, "math.", "ad"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	items := result.(CompletionList).Items
	item, ok := findCompletionItem(items, "add")
	if !ok {
		t.Fatalf("missing package member add in %#v", items)
	}
	if item.InsertText != "add" {
		t.Fatalf("add insertText = %q, want add", item.InsertText)
	}
	if item.Detail != "function from math" {
		t.Fatalf("add detail = %q, want function from math", item.Detail)
	}
	assertMissingCompletion(t, items, "append")
	assertMissingCompletion(t, items, "helper")
	assertMissingCompletion(t, items, "total")
	assertMissingCompletion(t, items, "hidden")
	assertMissingCompletion(t, items, "trim")
	assertMissingCompletion(t, items, "strings")
}

func TestCompletionOmitsPrivateSamePackageMembersAfterDot(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.tx")
	betaPath := filepath.Join(dir, "beta.tx")
	beta2Path := filepath.Join(dir, "beta2.tx")
	mainSrc := `package main
import "beta.tx"

func main() int {
    return beta.
}
`
	betaSrc := `package beta
pub func beta() int {
    return value2()
}
`
	beta2Src := `package beta
func value2() int {
    return 2
}
`
	for path, src := range map[string]string{
		mainPath:  mainSrc,
		betaPath:  betaSrc,
		beta2Path: beta2Src,
	} {
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mainURI := uriFromPath(normalizePath(mainPath))
	server.documents[mainURI] = mainSrc

	result, err := server.handleCompletion(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": mainURI},
		"position":     positionOfAfter(t, mainSrc, "return beta.", ""),
	}))
	if err != nil {
		t.Fatal(err)
	}

	items := result.(CompletionList).Items
	if _, ok := findCompletionItem(items, "beta"); !ok {
		t.Fatalf("missing public package member beta in %#v", items)
	}
	assertMissingCompletion(t, items, "value2")
}

func TestCompletionFiltersStandardLibraryMembersAfterDot(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	uri := "file:///tmp/main.tx"
	src := `package main
import "io"
import "csv"

func main() int {
    io.re
    return 0
}
`
	server.documents[uri] = src

	result, err := server.handleCompletion(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     positionOfAfter(t, src, "io.", "re"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	items := result.(CompletionList).Items
	if item, ok := findCompletionItem(items, "readLine"); !ok {
		t.Fatalf("missing package member readLine in %#v", items)
	} else if item.Detail != "io.readLine() string" {
		t.Fatalf("readLine detail = %q, want io.readLine() string", item.Detail)
	}
	if _, ok := findCompletionItem(items, "readFile"); !ok {
		t.Fatalf("missing package member readFile in %#v", items)
	}
	assertMissingCompletion(t, items, "write")
	assertMissingCompletion(t, items, "csv")
}

func TestCompletionImportsMissingStandardLibraryMembersAfterDot(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	uri := "file:///tmp/main.tx"
	src := `package main

func main() int {
    io.re
    return 0
}
`
	server.documents[uri] = src

	result, err := server.handleCompletion(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     positionOfAfter(t, src, "io.", "re"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	items := result.(CompletionList).Items
	if item, ok := findCompletionItem(items, "readLine"); !ok {
		t.Fatalf("missing package member readLine in %#v", items)
	} else {
		assertCompletionImportEdit(t, item, 1, `import "io"`+"\n")
	}
	if item, ok := findCompletionItem(items, "readFile"); !ok {
		t.Fatalf("missing package member readFile in %#v", items)
	} else if item.InsertText != "readFile" {
		t.Fatalf("readFile insertText = %q, want readFile", item.InsertText)
	}
	assertMissingCompletion(t, items, "print")
	assertMissingCompletion(t, items, "io")
}

func TestCompletionUsesDeclaredPackageName(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.tx")
	libPath := filepath.Join(dir, "calc.tx")
	mainSrc := `package main
import "calc.tx"

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

	items := result.(CompletionList).Items
	if _, ok := findCompletionItem(items, "math"); !ok {
		t.Fatalf("missing declared package completion math in %#v", items)
	}
	if _, ok := findCompletionItem(items, "math.add"); !ok {
		t.Fatalf("missing declared package function completion math.add in %#v", items)
	}
	assertMissingCompletion(t, items, "calc")
	assertMissingCompletion(t, items, "calc.add")
}

func TestCompletionIncludesSourceAutoImportFunctions(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.tx")
	mathPath := filepath.Join(dir, "math.tx")
	mainSrc := `package main

func main() int {
    return
}
`
	mathSrc := `package math
pub func add(a int, b int) int {
    return a + b
}

func hidden() int {
    return 0
}
`
	if err := os.WriteFile(mathPath, []byte(mathSrc), 0o644); err != nil {
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

	items := result.(CompletionList).Items
	item, ok := findCompletionItem(items, "math.add")
	if !ok {
		t.Fatalf("missing source auto-import completion math.add in %#v", items)
	}
	if item.InsertText != "math.add" {
		t.Fatalf("math.add insertText = %q, want math.add", item.InsertText)
	}
	if item.FilterText != "math.add add" {
		t.Fatalf("math.add filterText = %q, want math.add add", item.FilterText)
	}
	assertCompletionImportEdit(t, item, 1, `import "math.tx"`+"\n")
	assertMissingCompletion(t, items, "math.hidden")
	assertMissingCompletion(t, items, "hidden")
}

func TestCompletionSourceAutoImportUsesDeclaredPackageName(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.tx")
	calcPath := filepath.Join(dir, "calc.tx")
	mainSrc := `package main

func main() int {
    return
}
`
	calcSrc := `package math
pub func add(a int, b int) int {
    return a + b
}
`
	if err := os.WriteFile(calcPath, []byte(calcSrc), 0o644); err != nil {
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

	items := result.(CompletionList).Items
	item, ok := findCompletionItem(items, "math.add")
	if !ok {
		t.Fatalf("missing source auto-import completion math.add in %#v", items)
	}
	assertCompletionImportEdit(t, item, 1, `import "calc.tx"`+"\n")
	assertMissingCompletion(t, items, "calc.add")
}

func TestCompletionImportsMissingSourceMembersAfterDot(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.tx")
	mathPath := filepath.Join(dir, "math.tx")
	mainSrc := `package main

func main() int {
    return math.a
}
`
	mathSrc := `package math
pub func add(a int, b int) int {
    return a + b
}
`
	if err := os.WriteFile(mathPath, []byte(mathSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	mainURI := uriFromPath(normalizePath(mainPath))
	server.documents[mainURI] = mainSrc

	result, err := server.handleCompletion(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": mainURI},
		"position":     positionOfAfter(t, mainSrc, "math.", "a"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	items := result.(CompletionList).Items
	item, ok := findCompletionItem(items, "add")
	if !ok {
		t.Fatalf("missing source member auto-import completion add in %#v", items)
	}
	assertCompletionImportEdit(t, item, 1, `import "math.tx"`+"\n")
	assertMissingCompletion(t, items, "math.add")
	assertMissingCompletion(t, items, "print")
}

func TestCompletionIncludesSamePackageSiblingExports(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	dir := t.TempDir()
	mathDir := filepath.Join(dir, "math")
	if err := os.MkdirAll(mathDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(dir, "main.tx")
	addPath := filepath.Join(mathDir, "add.tx")
	doublePath := filepath.Join(mathDir, "double.tx")
	mainSrc := `package main
import "math/add.tx"

func main() int {
    return
}
`
	addSrc := `package math
pub func add(a int, b int) int {
    return a + b
}
`
	doubleSrc := `package math
pub func double(n int) int {
    return n + n
}
`
	for path, src := range map[string]string{
		addPath:    addSrc,
		doublePath: doubleSrc,
	} {
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
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

	items := result.(CompletionList).Items
	doubleItem, ok := findCompletionItem(items, "math.double")
	if !ok {
		t.Fatalf("missing same-package auto-import completion math.double in %#v", items)
	}
	if len(doubleItem.AdditionalTextEdits) != 0 {
		t.Fatalf("math.double additional edits = %#v, want none", doubleItem.AdditionalTextEdits)
	}

	addItem, ok := findCompletionItem(items, "math.add")
	if !ok {
		t.Fatalf("missing imported completion math.add in %#v", items)
	}
	if len(addItem.AdditionalTextEdits) != 0 {
		t.Fatalf("math.add additional edits = %#v, want none", addItem.AdditionalTextEdits)
	}
	if countCompletionLabels(items, "math.add") != 1 {
		t.Fatalf("math.add completion count = %d, want 1", countCompletionLabels(items, "math.add"))
	}
}

func TestCompletionKeepsAmbiguousSourceAutoImportsPathSpecific(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	dir := t.TempDir()
	firstDir := filepath.Join(dir, "first")
	secondDir := filepath.Join(dir, "second")
	if err := os.MkdirAll(firstDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(secondDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(dir, "main.tx")
	mainSrc := `package main

func main() int {
    return
}
`
	moduleSrc := `package math
pub func add(a int, b int) int {
    return a + b
}
`
	for _, path := range []string{filepath.Join(firstDir, "math.tx"), filepath.Join(secondDir, "math.tx")} {
		if err := os.WriteFile(path, []byte(moduleSrc), 0o644); err != nil {
			t.Fatal(err)
		}
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

	items := result.(CompletionList).Items
	if countCompletionLabels(items, "math.add") != 2 {
		t.Fatalf("math.add completion count = %d, want 2 in %#v", countCompletionLabels(items, "math.add"), items)
	}
	wantEdits := map[string]struct{}{
		`import "first/math.tx"` + "\n":  {},
		`import "second/math.tx"` + "\n": {},
	}
	for _, item := range items {
		if item.Label != "math.add" {
			continue
		}
		if len(item.AdditionalTextEdits) != 1 {
			t.Fatalf("math.add additional edits = %#v, want one", item.AdditionalTextEdits)
		}
		if _, ok := wantEdits[item.AdditionalTextEdits[0].NewText]; !ok {
			t.Fatalf("math.add import edit = %q, want one of %#v", item.AdditionalTextEdits[0].NewText, wantEdits)
		}
		delete(wantEdits, item.AdditionalTextEdits[0].NewText)
		if !strings.Contains(item.Detail, "math.tx") {
			t.Fatalf("math.add detail = %q, want path-specific detail", item.Detail)
		}
	}
	if len(wantEdits) != 0 {
		t.Fatalf("missing import edits %#v", wantEdits)
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

func TestHoverReturnsSamePackagePrivateFunctionSignature(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	dir := t.TempDir()
	betaPath := filepath.Join(dir, "beta.tx")
	beta2Path := filepath.Join(dir, "beta2.tx")
	betaSrc := `package beta
pub func beta() int {
    return value2()
}
`
	beta2Src := `package beta
func value2() int {
    return 2
}
`
	if err := os.WriteFile(betaPath, []byte(betaSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(beta2Path, []byte(beta2Src), 0o644); err != nil {
		t.Fatal(err)
	}
	betaURI := uriFromPath(normalizePath(betaPath))
	server.documents[betaURI] = betaSrc

	result, err := server.handleHover(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": betaURI},
		"position":     positionOfAfter(t, betaSrc, "return ", "value2"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	hover := result.(Hover)
	if hover.Contents.Value != "`func value2() int`" {
		t.Fatalf("hover = %#v, want same-package private function signature", hover)
	}
	assertRange(t, hover.Range, positionOfAfter(t, betaSrc, "return ", "value2"), "value2")
}

func TestHoverReturnsStandardLibraryFunctionSignature(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	uri := "file:///tmp/main.tx"
	src := `package main
import "csv"

func main() int {
    let cells list[string] = csv.read("input.csv", 2)
    return len(cells)
}
`
	server.documents[uri] = src

	result, err := server.handleHover(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     positionOfAfter(t, src, "csv.", "read"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	hover := result.(Hover)
	if hover.Contents.Value != "`csv.read(path string, columns int) list[string]`" {
		t.Fatalf("hover = %#v, want standard library function signature", hover)
	}
	assertRange(t, hover.Range, positionOfAfter(t, src, "csv.", "read"), "read")
}

func TestHoverReturnsGPULaunchSignature(t *testing.T) {
	server := NewServer(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{})
	uri := "file:///tmp/main.tx"
	src := `package main
import "gpu"

kernel func fill(out gpu.Buffer[int], n int) {
    let i int = gpu.globalX()
    if i < n {
        out[i] = 1
    }
}

func main() int {
    let out gpu.Buffer[int] = gpu.alloc(1)
    gpu.launch(fill, 1, 1, 1, 1, 1, 1, out, 1)
    return 0
}
`
	server.documents[uri] = src

	result, err := server.handleHover(mustJSON(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     positionOfAfter(t, src, "gpu.", "launch"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	hover := result.(Hover)
	want := "`gpu.launch(kernel, gridX int, gridY int, gridZ int, blockX int, blockY int, blockZ int, args...)`"
	if hover.Contents.Value != want {
		t.Fatalf("hover = %#v, want gpu.launch signature", hover)
	}
	assertRange(t, hover.Range, positionOfAfter(t, src, "gpu.", "launch"), "launch")
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

func countCompletionLabels(items []CompletionItem, label string) int {
	count := 0
	for _, item := range items {
		if item.Label == label {
			count++
		}
	}
	return count
}

func assertMissingCompletion(t *testing.T, items []CompletionItem, label string) {
	t.Helper()

	if item, ok := findCompletionItem(items, label); ok {
		t.Fatalf("completion %q = %#v, want missing", label, item)
	}
}

func assertCompletionTextEdit(t *testing.T, item CompletionItem, newText string) {
	t.Helper()

	if item.TextEdit == nil {
		t.Fatalf("completion %q text edit = nil, want %q", item.Label, newText)
	}
	if item.TextEdit.NewText != newText {
		t.Fatalf("completion %q text edit newText = %q, want %q", item.Label, item.TextEdit.NewText, newText)
	}
}

func assertCompletionImportEdit(t *testing.T, item CompletionItem, line int, newText string) {
	t.Helper()

	if len(item.AdditionalTextEdits) != 1 {
		t.Fatalf("completion %q additional edits = %#v, want one import edit", item.Label, item.AdditionalTextEdits)
	}
	edit := item.AdditionalTextEdits[0]
	wantRange := Range{
		Start: Position{Line: line, Character: 0},
		End:   Position{Line: line, Character: 0},
	}
	if edit.Range != wantRange {
		t.Fatalf("completion %q import edit range = %#v, want %#v", item.Label, edit.Range, wantRange)
	}
	if edit.NewText != newText {
		t.Fatalf("completion %q import edit newText = %q, want %q", item.Label, edit.NewText, newText)
	}
}
