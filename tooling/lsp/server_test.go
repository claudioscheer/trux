package main

import (
	"bufio"
	"bytes"
	"encoding/json"
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

func TestDiagnosticsSkipTypeCheckForImportedPrograms(t *testing.T) {
	diagnostics := diagnosticsFor("file:///tmp/main.tx", `package main
import "math.tx"

func main() int {
    return add(1, 2)
}`)

	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
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
