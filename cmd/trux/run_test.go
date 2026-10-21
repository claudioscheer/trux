package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	stdhttp "net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRunFileParsesValidSourceWithNoOutput(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
func main() int {
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err != nil {
		t.Fatal(err)
	}

	if out.String() != "" {
		t.Fatalf("output = %q, want empty output", out.String())
	}
}

func TestRunFileCompilesAndExecutesIntegerProgram(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
func add(a int, b int) int {
    return a + b
}

func main() int {
    let x int = add(1, 2)
    print(x)
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err != nil {
		t.Fatal(err)
	}

	if out.String() != "3\n" {
		t.Fatalf("output = %q, want 3\\n", out.String())
	}
}

func TestRunFileCompilesAndExecutesPrimitiveProgram(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
func label() string {
    return "trux"
}

func ready() bool {
    return true
}

func main() int {
    let name string = label()
    let ok bool = ready()
    print(name, " ", 1)
    print(ok)
    print("line\nquote: \"")
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err != nil {
		t.Fatal(err)
	}

	want := "trux 1\ntrue\nline\nquote: \"\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestRunFileKeepsBranchAssignedLoopStringAliveForCaller(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
func pick(flag bool) string {
    let acc string = ""
    let i int = 0
    for ; i < 4; i = i + 1 {
        if flag {
            acc = acc + "x"
        } else {
            acc = "ok"
        }
    }
    return acc
}

func main() int {
    let value string = pick(true)
    let junk string = "01230123012301230123012301230123" + "01230123012301230123012301230123"
    print(value)
    print(len(value))
    print(len(junk))
    let other string = pick(false)
    let junkOther string = "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz" + "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
    print(other)
    print(len(junkOther))
    return 0
}`)

	var out bytes.Buffer
	if err := runFile(&out, path); err != nil {
		t.Fatal(err)
	}
	want := "xxxx\n4\n64\nok\n64\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestRunFileKeepsJumpAssignedLoopStringAliveForCaller(t *testing.T) {
	requireCC(t)

	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "continue",
			src: `package main
func cont(flag bool) string {
    let acc string = ""
    let i int = 0
    for ; i < 4; i = i + 1 {
        if flag {
            acc = acc + "x"
            continue
            acc = "ok"
        }
        acc = "no"
    }
    return acc
}

func main() int {
    let value string = cont(true)
    let junk string = "01230123012301230123012301230123" + "01230123012301230123012301230123"
    print(value)
    print(len(value))
    print(len(junk))
    return 0
}`,
			want: "xxxx\n4\n64\n",
		},
		{
			name: "break",
			src: `package main
func broken(flag bool) string {
    let acc string = ""
    let i int = 0
    for ; i < 4; i = i + 1 {
        if flag {
            acc = acc + "x"
            break
            acc = "ok"
        }
        acc = "no"
    }
    return acc
}

func main() int {
    let value string = broken(true)
    let junk string = "0" + "123"
    print(value)
    print(len(value))
    print(len(junk))
    return 0
}`,
			want: "x\n1\n4\n",
		},
		{
			name: "break skips post",
			src: `package main
func broken(flag bool) string {
    let acc string = ""
    let i int = 0
    for ; i < 4; acc = "ok" {
        if flag {
            acc = acc + "x"
            break
        }
        i = i + 1
    }
    return acc
}

func main() int {
    let value string = broken(true)
    let junk string = "0" + "123"
    print(value)
    print(len(value))
    print(len(junk))
    return 0
}`,
			want: "x\n1\n4\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempSource(t, tt.src)
			var out bytes.Buffer
			if err := runFile(&out, path); err != nil {
				t.Fatal(err)
			}
			if out.String() != tt.want {
				t.Fatalf("output = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestRunFileKeepsTempStringReturnAliveForCaller(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
func bang(name string) string {
    let s string = name + "!"
    return s
}

func main() int {
    print(bang("a"))
    print(bang("b"))
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err != nil {
		t.Fatal(err)
	}

	if out.String() != "a!\nb!\n" {
		t.Fatalf("output = %q, want a!\\nb!\\n", out.String())
	}
}

func TestRunFileCompilesAndExecutesControlAndStringExamples(t *testing.T) {
	requireCC(t)

	tests := []struct {
		path string
		want string
	}{
		{
			path: "../../examples/control_flow.tx",
			want: "big\n0\n1\n2\ndone\n",
		},
		{
			path: "../../examples/else_if.tx",
			want: "grade B\n",
		},
		{
			path: "../../examples/primitives.tx",
			want: "trux 3 false\n2\nline\nquote: \"\n",
		},
		{
			path: "../../examples/fibonacci.tx",
			want: "iterative\n0\n1\n1\n2\n3\n5\n8\n13\n21\n34\nrecursive\n0\n1\n1\n2\n3\n5\n8\n13\n21\n34\n",
		},
		{
			path: "../../examples/loops.tx",
			want: "16 10\n16 10\n",
		},
		{
			path: "../../examples/return_paths.tx",
			want: "42\n7\n4\n0\n",
		},
		{
			path: "../../examples/strings.tx",
			want: "trux compiler\nhello, trux compiler\ntrux compiler!\nempty\ntrue\ntrue\ntrue\ntrue\nfalse\ntrue\ntrue\ntrue\n",
		},
		{
			path: "../../examples/assertions.tx",
			want: "mean 9\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			var out bytes.Buffer
			err := runFile(&out, tt.path)
			if err != nil {
				t.Fatal(err)
			}
			if out.String() != tt.want {
				t.Fatalf("output = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestRunFileCompilesAndExecutesCollectionExamples(t *testing.T) {
	requireCC(t)

	tests := []struct {
		path string
		want string
	}{
		{
			path: "../../examples/collections.tx",
			want: "3\n1 3\n9 2\n8 8 2\n5 0 7\nr ru 4\n2 2\n3 2 3\n8 2\n",
		},
		{
			path: "../../examples/ownership_clone.tx",
			want: "99 2\n20 30\n1 42\n7 9\nab cd 2\n",
		},
		{
			path: "../../examples/mut_parameters.tx",
			want: "2 2 2\n1 hello trux\n8 8\n9 9\n1 2 2\n10 20 0\n",
		},
		{
			path: "../../examples/matrix_multiply_2x2.tx",
			want: "19 22\n43 50\n19 22\n43 50\n",
		},
		{
			path: "../../examples/jagged_lists.tx",
			want: "3 3 1\n12 12 6\n",
		},
		{
			path: "../../examples/nested_slices.tx",
			want: "1 2 3\n2.5 9.5\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			var out bytes.Buffer
			err := runFile(&out, tt.path)
			if err != nil {
				t.Fatal(err)
			}
			if out.String() != tt.want {
				t.Fatalf("output = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestRunFileCompilesAndExecutesModuleExamples(t *testing.T) {
	requireCC(t)

	tests := []struct {
		path string
		want string
	}{
		{
			path: "../../examples/modules/basic/main.tx",
			want: "7\n",
		},
		{
			path: "../../examples/modules/transitive/main.tx",
			want: "hello modules 42\n",
		},
		{
			path: "../../examples/modules/private_names/main.tx",
			want: "11 42\n",
		},
		{
			path: "../../examples/modules/same_package_math/main.tx",
			want: "7 25 12\n",
		},
		{
			path: "../../examples/modules/mut_parameters/main.tx",
			want: "2 20\n4 4\n7 7\n1 2 2\n10 20 0\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			var out bytes.Buffer
			err := runFile(&out, tt.path)
			if err != nil {
				t.Fatal(err)
			}
			if out.String() != tt.want {
				t.Fatalf("output = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestBuildCommandCompilesAndRunsHTTPServer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("http runtime is POSIX-only")
	}
	requireCC(t)

	port := freeTCPPort(t)
	path := writeTempSource(t, fmt.Sprintf(`package main
import "http"

func handle(request int) int {
    let method string = http.method(request)
    let path string = http.path(request)
    let query string = http.query(request)
    let agent string = http.header(request, "User-Agent")
    let body string = http.body(request)

    if method == "POST" {
        if path == "/hello" {
            http.respond(request, 200, "text/plain", query + "|" + agent + "|" + body)
            return 0
        }
    }

    http.respond(request, 404, "text/plain", "missing")
    return 0
}

func main() int {
    http.serve("127.0.0.1", %d, 4, handle)
    return 0
}`, port))
	outputPath := filepath.Join(t.TempDir(), "server")
	if err := buildFile(path, outputPath); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(outputPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	})

	client := &stdhttp.Client{Timeout: 500 * time.Millisecond}
	url := fmt.Sprintf("http://127.0.0.1:%d/hello?name=trux", port)
	var resp *stdhttp.Response
	var err error
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		req, reqErr := stdhttp.NewRequest("POST", url, strings.NewReader("payload"))
		if reqErr != nil {
			t.Fatal(reqErr)
		}
		req.Header.Set("User-Agent", "trux-test")
		resp, err = client.Do(req)
		if err == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("HTTP server did not respond: %v\nstderr:\n%s", err, stderr.String())
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200, body = %q", resp.StatusCode, string(body))
	}
	if string(body) != "name=trux|trux-test|payload" {
		t.Fatalf("body = %q, want request fields", string(body))
	}

	rawHTTPStatus := func(t *testing.T, request string) string {
		t.Helper()
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 500*time.Millisecond)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		if err := conn.SetDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
			t.Fatal(err)
		}
		if _, err := conn.Write([]byte(request)); err != nil {
			t.Fatal(err)
		}
		response, err := io.ReadAll(conn)
		if err != nil {
			t.Fatal(err)
		}
		line, _, _ := strings.Cut(string(response), "\r\n")
		return line
	}

	duplicateCL := "POST /hello HTTP/1.1\r\nHost: 127.0.0.1\r\nContent-Length: 5\r\nContent-Length: 7\r\n\r\npayload"
	if status := rawHTTPStatus(t, duplicateCL); status != "HTTP/1.1 400 Bad Request" {
		t.Fatalf("duplicate Content-Length status = %q, want 400", status)
	}

	transferEncoding := "POST /hello HTTP/1.1\r\nHost: 127.0.0.1\r\nTransfer-Encoding: gzip, chunked\r\nContent-Length: 7\r\n\r\npayload"
	if status := rawHTTPStatus(t, transferEncoding); status != "HTTP/1.1 400 Bad Request" {
		t.Fatalf("Transfer-Encoding status = %q, want 400", status)
	}
}

func TestRunFileCompilesAndExecutesNestedCollections(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
func buildRows() list[list[int]] {
    let rows list[list[int]] = list[list[int]]{list[int]{1, 2}, list[int]{3}}
    append(rows, list[int]{4, 5, 6})
    return rows
}

func main() int {
    let rows list[list[int]] = buildRows()
    let copied list[list[int]] = clone(rows)
    append(copied[0], 9)

    let fixed [2][2]int = [2][2]int{[2]int{1, 2}, [2]int{3, 4}}

    let slices [][]float = make([][]float, 2)
    slices[0] = make([]float, 2)
    slices[0][1] = 2.5

    print(len(rows[0]), " ", len(copied[0]), " ", copied[0][2], " ", rows[2][2])
    print(fixed[1][1], " ", slices[0][1])
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err != nil {
		t.Fatal(err)
	}

	want := "2 3 9 6\n4 2.5\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestRunFileCompilesAndExecutesOwnershipCopyOut(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
func middle() []int {
    let xs [3]int = [3]int{1, 2, 3}
    return xs[1:2]
}

func build() list[string] {
    let xs list[string] = list[string]{}
    append(xs, "a" + "b")
    return xs
}

func mid(xs []int) []int {
    return xs[1:2]
}

func midOwned(xs []int) []int {
    return clone(xs[1:2])
}

func frameOwnedLocal() []int {
    let xs [3]int = [3]int{1, 2, 3}
    let ys []int = clone(xs[:])
    return ys
}

func main() int {
    let local []int = middle()
    print(local[0])

    let words list[string] = build()
    print(words[0], " ", len(words))

    let xs [3]int = [3]int{1, 2, 3}
    let borrowed []int = mid(xs[:])
    let owned []int = midOwned(xs[:])
    xs[1] = 9
    print(borrowed[0], " ", owned[0])

    let framed []int = frameOwnedLocal()
    print(framed[0], " ", framed[2])
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err != nil {
		t.Fatal(err)
	}

	want := "2\nab 1\n9 2\n1 3\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestRunFileCompilesAndExecutesModuleProgram(t *testing.T) {
	requireCC(t)

	dir := t.TempDir()
	mainPath := writeTempFile(t, dir, "main.tx", `package main
import "math.tx"
import "words.tx"

func main() int {
    print(words.label(), " ", math.add(2, 5))
    return 0
}`)
	writeTempFile(t, dir, "math.tx", `package math
func base() int {
    return 2
}

pub func add(a int, b int) int {
    return a + b + base()
}`)
	writeTempFile(t, dir, "words.tx", `package words
pub func label() string {
    return "module"
}`)

	var out bytes.Buffer
	err := runFile(&out, mainPath)
	if err != nil {
		t.Fatal(err)
	}

	if out.String() != "module 9\n" {
		t.Fatalf("output = %q, want module 9\\n", out.String())
	}
}

func TestRunFileCompilesAndExecutesInputProgram(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
import "io"

func main() int {
    let name string = io.readLine()
    let count int = io.readInt()
    let ratio float = io.readFloat()
    let ready bool = io.readBool()
    print(name, " ", count + 1, " ", ratio, " ", ready)
    return 0
}`)

	var out bytes.Buffer
	err := runFileWithOptions(&out, path, runOptions{Stdin: strings.NewReader("Ada\n41\n2.5\ntrue\n")})
	if err != nil {
		t.Fatal(err)
	}

	if out.String() != "Ada 42 2.5 true\n" {
		t.Fatalf("output = %q, want Ada 42 2.5 true\\n", out.String())
	}
}

func TestRunFileCompilesAndExecutesPerformanceExample(t *testing.T) {
	requireCC(t)

	var out bytes.Buffer
	err := runFile(&out, "../../examples/performance_integer.tx")
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("output = %q, want checksum and elapsed lines", out.String())
	}
	if lines[0] != "workload checksum 496477" {
		t.Fatalf("checksum line = %q, want stable checksum", lines[0])
	}
	if !strings.HasPrefix(lines[1], "elapsed nanos ") {
		t.Fatalf("elapsed line = %q, want elapsed nanos prefix", lines[1])
	}

	elapsed, err := strconv.ParseInt(strings.TrimPrefix(lines[1], "elapsed nanos "), 10, 64)
	if err != nil {
		t.Fatalf("parse elapsed nanos: %v", err)
	}
	if elapsed <= 0 {
		t.Fatalf("elapsed nanos = %d, want positive duration", elapsed)
	}
}

func TestRunFileCompilesAndExecutesFileAndCSVIO(t *testing.T) {
	requireCC(t)

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.txt")
	copyPath := filepath.Join(dir, "copy.txt")
	csvPath := filepath.Join(dir, "input.csv")
	outCSVPath := filepath.Join(dir, "out.csv")
	if err := os.WriteFile(inputPath, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(csvPath, []byte("name,score\n\"A, B\",2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	path := writeTempSource(t, `package main
import "io"
import "csv"

func main() int {
    let text string, err1 error = io.readFile(`+truxString(inputPath)+`)
    assert(err1 == nil, "read input.txt")
    let err2 error = io.writeFile(`+truxString(copyPath)+`, text + "!")
    assert(err2 == nil, "write copy.txt")

    let cells list[string], err3 error = csv.read(`+truxString(csvPath)+`, 2)
    assert(err3 == nil, "read csv")
    append(cells, "Cara")
    append(cells, "3")
    let err4 error = csv.write(`+truxString(outCSVPath)+`, cells, 2)
    assert(err4 == nil, "write csv")

    print(text)
    print(len(cells), " ", cells[2])
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err != nil {
		t.Fatal(err)
	}

	if out.String() != "hello\n6 A, B\n" {
		t.Fatalf("output = %q, want file/csv output", out.String())
	}
	if got := readFile(t, copyPath); got != "hello!" {
		t.Fatalf("copy file = %q, want hello!", got)
	}
	if got := readFile(t, outCSVPath); got != "name,score\n\"A, B\",2\nCara,3\n" {
		t.Fatalf("written csv = %q, want appended csv", got)
	}
}

func TestRunFileCompilesAndExecutesMultiReturnAndErrors(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
import "errors"
import "io"

func divmod(a int, b int) (int, int) {
    return a / b, a % b
}

func forwardDivmod(a int, b int) (int, int) {
    return divmod(a, b)
}

func makeMessage(id int) (string, error) {
    return "message " + "from id", nil
}

func forwardMessage(id int) (string, error) {
    return makeMessage(id)
}

func parsePositive(n int) (int, error) {
    if n < 0 {
        return 0, errors.new("negative number")
    }
    return n, nil
}

func main() int {
    let q int, r int = divmod(14, 3)
    print(q, " ", r)

    let _, rem int = divmod(14, 3)
    print("rem=", rem)

    let quot int, _ int = divmod(14, 3)
    print("quot=", quot)

    q, r = forwardDivmod(20, 6)
    print("forward: ", q, " ", r)

    let msg string, msgErr error = forwardMessage(42)
    assert(msgErr == nil, "forwardMessage error")
    print(msg)

    _, r = divmod(25, 4)
    print("r=", r)

    let val int, err error = parsePositive(10)
    if err != nil {
        print("error: ", errors.message(err))
    } else {
        print("ok ", val)
    }

    val, err = parsePositive(-5)
    if err != nil {
        print("error: ", errors.message(err))
        print(err)
    }

    let nilErr error = nil
    print(nilErr)
    if nilErr == nil {
        print("nilErr is nil")
    }

    let _, ioErr error = io.readFile("non_existent_file_path_12345.xyz")
    if ioErr != nil {
        print("caught io error: ", errors.message(ioErr))
    }

    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err != nil {
		t.Fatal(err)
	}

	want := "4 2\n" +
		"rem=2\n" +
		"quot=4\n" +
		"forward: 3 2\n" +
		"message from id\n" +
		"r=1\n" +
		"ok 10\n" +
		"error: negative number\n" +
		"negative number\n" +
		"<nil>\n" +
		"nilErr is nil\n" +
		"caught io error: failed to open file for reading\n"

	if out.String() != want {
		t.Fatalf("output =\n%q\nwant:\n%q", out.String(), want)
	}
}

func TestRunFileReturnsRuntimeBoundsError(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
func main() int {
    let xs [1]int = [1]int{1}
    print(xs[1])
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err == nil {
		t.Fatal("expected error")
	}
	if out.String() != "" {
		t.Fatalf("output = %q, want empty output", out.String())
	}
	if !strings.Contains(err.Error(), "trux runtime error: index 1 out of bounds for length 1") {
		t.Fatalf("error = %q, want bounds error", err.Error())
	}
}

func TestRunFileRewindsLoopTempsAndKeepsEscapes(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
func temps(n int) int {
    let i int = 0
    let count int = 0
    for ; i < n; i = i + 1 {
        let temp string = "a" + "b"
        if temp == "ab" {
            count = count + 1
        }
    }
    return count
}

func kept(n int) string {
    let acc string = ""
    let i int = 0
    for ; i < n; i = i + 1 {
        acc = acc + "x"
    }
    return acc
}

func collect(n int) int {
    let words list[string] = list[string]{}
    let i int = 0
    for ; i < n; i = i + 1 {
        append(words, "a" + "b")
    }
    return len(words)
}

func word() string {
    return "a" + "b"
}

func countWords(n int) int {
    let i int = 0
    let count int = 0
    for ; i < n; i = i + 1 {
        let temp string = word()
        if temp == "ab" {
            count = count + 1
        }
    }
    return count
}

func pushWord(mut words list[string], word string) int {
    append(words, word)
    return len(words)
}

func fill(n int) int {
    let words list[string] = list[string]{}
    let i int = 0
    for ; i < n; i = i + 1 {
        pushWord(words, "a" + "b")
    }
    print(words[0], " ", words[2])
    return len(words)
}

func early() string {
    let i int = 0
    for ; i < 5; i = i + 1 {
        let temp string = "h" + "i"
        if i == 3 {
            return temp
        }
    }
    return ""
}

func leaf(n int) int {
    if n == 0 {
        return 2
    }
    let i int = 0
    let sum int = 0
    for ; i < 2; i = i + 1 {
        let inner string = "v" + "w"
        sum = sum + leaf(n - 1) + len(inner)
    }
    return sum
}

func sumViews(n int) int {
    let xs [3]int = [3]int{1, 2, 3}
    let i int = 0
    let total int = 0
    for ; i < n; i = i + 1 {
        let view []int = clone(xs[:])
        total = total + view[0] + view[2]
    }
    return total
}

func keptSlice() int {
    let xs [3]int = [3]int{4, 5, 6}
    let view []int = xs[:]
    let i int = 0
    for ; i < 3; i = i + 1 {
        view = clone(xs[:])
    }
    return view[0] + view[2]
}

func mix(n int) int {
    let nums list[int] = list[int]{}
    let i int = 0
    for ; i < n; i = i + 1 {
        let temp string = "a" + "b"
        append(nums, len(temp))
    }
    return nums[0] + nums[n - 1] + len(nums)
}

func hot(n int) int {
    let checksum int = 1
    for let i int = 0; i < n; i = i + 1 {
        checksum = checksum + i
    }
    return checksum
}

func main() int {
    print(temps(20))
    print(kept(4))
    print(collect(3))
    print(countWords(4))
    print(fill(3))
    print(early())
    print(leaf(2))
    print(sumViews(3))
    print(keptSlice())
    print(mix(4))
    print(hot(5))
    return 0
}`)
	var out bytes.Buffer
	if err := runFile(&out, path); err != nil {
		t.Fatal(err)
	}
	want := "20\nxxxx\n3\n4\nab ab\n3\nhi\n20\n12\n10\n8\n11\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestRunFileReusesFrameChunksAcrossCalls(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
func bang(value string) string {
    return value + "!"
}

func main() int {
    let i int = 0
    let word string = "a"
    for ; i < 5; i = i + 1 {
        word = bang(word)
    }
    print(word)
    return 0
}`)
	var out bytes.Buffer
	if err := runFile(&out, path); err != nil {
		t.Fatal(err)
	}
	if out.String() != "a!!!!!\n" {
		t.Fatalf("output = %q, want a!!!!!", out.String())
	}
}

func TestRunFileBreakAndContinue(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
import "io"

func main() int {
    let i int = 0
    let seen int = 0
    for ; i < 5; i = i + 1 {
        if i == 2 {
            break
        }
        seen = seen + 1
    }
    print(seen)
    print(i)

    let j int = 0
    let kept int = 0
    for ; j < 4; j = j + 1 {
        if j == 1 {
            continue
        }
        kept = kept + 1
    }
    print(kept)
    print(j)

    let n int = 0
    for let x int = 0; x < 3; x = x + 1 {
        for let y int = 0; y < 3; y = y + 1 {
            if y == 1 {
                break
            }
            n = n + 1
        }
    }
    print(n)

    let m int = 0
    for let x int = 0; x < 2; x = x + 1 {
        for let y int = 0; y < 3; y = y + 1 {
            if y == 1 {
                continue
            }
            m = m + 1
        }
    }
    print(m)

    let k int = 0
    for ; io.readInt() == 1; k = k + 1 {
        if k == 0 {
            continue
        }
    }
    print(k)
    return 0
}`)

	var out bytes.Buffer
	err := runFileWithOptions(&out, path, runOptions{Stdin: strings.NewReader("1\n1\n0\n")})
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != "2\n2\n3\n4\n3\n4\n2\n" {
		t.Fatalf("output = %q, want break and continue counts", out.String())
	}
}

func TestCompileFileEmitsBreakAndContinueInKernel(t *testing.T) {
	path := writeTempSource(t, `package main
import "gpu"

kernel func fill(out gpu.Buffer[int], n int) {
    for let i int = 0; i < n; i = i + 1 {
        if i == 1 {
            continue
        }
        if i == 3 {
            break
        }
        out[0] = i
    }
}

func main() int {
    return 0
}`)
	result, err := compileFile(path, compileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.CSource, "break;") || !strings.Contains(result.CSource, "goto trux_continue_") || !strings.Contains(result.CSource, "trux_continue_0:") {
		t.Fatalf("generated source missing loop control:\n%s", result.CSource)
	}
}

func TestRunFileShortCircuitsLogicalOperators(t *testing.T) {
	requireCC(t)

	tests := []struct {
		name  string
		src   string
		stdin string
		want  string
	}{
		{
			name: "and skips read",
			src: `package main
import "io"
func main() int {
    print(io.readInt() == 1 && io.readInt() == 2)
    return 0
}`,
			stdin: "0\n",
			want:  "false\n",
		},
		{
			name: "or skips read",
			src: `package main
import "io"
func main() int {
    print(io.readInt() == 1 || io.readInt() == 2)
    return 0
}`,
			stdin: "1\n",
			want:  "true\n",
		},
		{
			name: "and reads both",
			src: `package main
import "io"
func main() int {
    print(io.readInt() == 1 && io.readInt() == 2)
    return 0
}`,
			stdin: "1\n9\n",
			want:  "false\n",
		},
		{
			name: "or reads both",
			src: `package main
import "io"
func main() int {
    print(io.readInt() == 0 || io.readInt() == 2)
    return 0
}`,
			stdin: "5\n2\n",
			want:  "true\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempSource(t, tt.src)
			var out bytes.Buffer
			err := runFileWithOptions(&out, path, runOptions{Stdin: strings.NewReader(tt.stdin)})
			if err != nil {
				t.Fatal(err)
			}
			if out.String() != tt.want {
				t.Fatalf("output = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestCompileFileEmitsLogicalOperatorsInKernel(t *testing.T) {
	path := writeTempSource(t, `package main
import "gpu"

kernel func fill(out gpu.Buffer[int], n int, enabled bool) {
    if enabled && n > 0 || !enabled {
        out[0] = 1
    }
}

func main() int {
    return 0
}`)
	result, err := compileFile(path, compileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.CSource, "&&") || !strings.Contains(result.CSource, "||") {
		t.Fatalf("generated source missing logical operators:\n%s", result.CSource)
	}
}

func TestRunFileCompilesAndExecutesRemainder(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
func main() int {
    print(17 % 5)
    print((10 % 3) * 2)
    return 0
}`)
	var out bytes.Buffer
	if err := runFile(&out, path); err != nil {
		t.Fatal(err)
	}
	if out.String() != "2\n2\n" {
		t.Fatalf("output = %q, want 2 and 2", out.String())
	}
}

func TestRunFileRemainderRejectsZeroAndOverflow(t *testing.T) {
	requireCC(t)

	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "zero",
			src: `package main
func main() int {
    print(10 % 0)
    return 0
}`,
			want: "trux runtime error: integer remainder by zero",
		},
		{
			name: "overflow",
			src: `package main
func main() int {
    let min int = (0 - 9223372036854775807) - 1
    let minusOne int = 0 - 1
    print(min % minusOne)
    return 0
}`,
			want: "trux runtime error: integer overflow in %",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempSource(t, tt.src)
			var out bytes.Buffer
			err := runFile(&out, path)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %s", err, tt.want)
			}
		})
	}
}

func TestCompileFileEmitsRemainderInKernel(t *testing.T) {
	path := writeTempSource(t, `package main
import "gpu"

kernel func fill(out gpu.Buffer[int], n int) {
    out[0] = n % 3
}

func main() int {
    return 0
}`)
	result, err := compileFile(path, compileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.CSource, "rt_rem_i64(") {
		t.Fatalf("generated source missing remainder helper:\n%s", result.CSource)
	}
}

func TestRunFileCompilesAndExecutesPrefixOperators(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
func main() int {
    let n int = -4
    let f float = -1.5
    let flag bool = !false
    let grouped bool = !(1 == 2)
    print(n)
    print(f)
    print(flag)
    print(grouped)
    return 0
}`)

	var out bytes.Buffer
	if err := runFile(&out, path); err != nil {
		t.Fatal(err)
	}
	if out.String() != "-4\n-1.5\ntrue\ntrue\n" {
		t.Fatalf("output = %q, want prefix results", out.String())
	}
}

func TestRunFileNegatingMinimumIntOverflows(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
func main() int {
    let min int = (0 - 9223372036854775807) - 1
    print(-min)
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err == nil {
		t.Fatal("expected overflow")
	}
	if !strings.Contains(err.Error(), "trux runtime error: integer overflow in -") {
		t.Fatalf("error = %q, want integer overflow", err.Error())
	}
}

func TestCompileFileEmitsPrefixOperatorsForHostAndKernel(t *testing.T) {
	path := writeTempSource(t, `package main
import "gpu"

kernel func scale(out gpu.Buffer[int], n int, enabled bool) {
    let i int = -gpu.globalX()
    if !enabled {
        out[0] = i
    }
}

func main() int {
    let n int = -1
    let f float = -1.5
    let flag bool = !false
    print(n)
    print(f)
    print(flag)
    return 0
}`)

	result, err := compileFile(path, compileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"rt_sub_i64(0,", "!(", "-(", "__global__"} {
		if !strings.Contains(result.CSource, part) {
			t.Fatalf("generated source missing %q:\n%s", part, result.CSource)
		}
	}
}

func TestRunFileReturnsRuntimeIntegerDivisionErrors(t *testing.T) {
	requireCC(t)

	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "division by zero",
			src: `package main
func main() int {
    print(10 / 0)
    return 0
}`,
			want: "trux runtime error: integer division by zero",
		},
		{
			name: "division overflow",
			src: `package main
func main() int {
    let min int = (0 - 9223372036854775807) - 1
    let minusOne int = 0 - 1
    print(min / minusOne)
    return 0
}`,
			want: "trux runtime error: integer overflow in /",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempSource(t, tt.src)

			var out bytes.Buffer
			err := runFile(&out, path)
			if err == nil {
				t.Fatal("expected error")
			}
			if out.String() != "" {
				t.Fatalf("output = %q, want empty output", out.String())
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestRunFileEvaluatesSideEffectingExpressionsLeftToRight(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
import "io"

func sub(a int, b int) int {
    return a - b
}

func main() int {
    print(sub(io.readInt(), io.readInt()))
    print(io.readInt() - io.readInt())
    return 0
}`)

	var out bytes.Buffer
	err := runFileWithOptions(&out, path, runOptions{Stdin: strings.NewReader("10\n3\n20\n8\n")})
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != "7\n12\n" {
		t.Fatalf("output = %q, want ordered reads", out.String())
	}
}

func TestRunFileAssertTrueContinuesWithoutEvaluatingMessage(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
func message() string {
    print("message evaluated")
    return "bad"
}

func main() int {
    assert(1 == 1, message())
    print("after")
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != "after\n" {
		t.Fatalf("output = %q, want after\\n", out.String())
	}
}

func TestRunFileAssertFalseCrashesApp(t *testing.T) {
	requireCC(t)

	path := writeTempSource(t, `package main
func main() int {
    print("before")
    assert(1 == 2, "numbers should match")
    print("after")
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err == nil {
		t.Fatal("expected error")
	}
	if out.String() != "before\n" {
		t.Fatalf("output = %q, want before\\n", out.String())
	}
	if !strings.Contains(err.Error(), "trux runtime error: assertion failed: numbers should match") {
		t.Fatalf("error = %q, want assertion failure", err.Error())
	}
}

func TestRunFileReturnsMissingImportWithSourceContext(t *testing.T) {
	dir := t.TempDir()
	mainPath := writeTempFile(t, dir, "main.tx", `package main
import "missing.tx"

func main() int {
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, mainPath)
	if err == nil {
		t.Fatal("expected error")
	}
	if out.String() != "" {
		t.Fatalf("output = %q, want empty output", out.String())
	}

	wantParts := []string{
		mainPath + ":2:1: cannot find module \"missing.tx\"",
		"1 | package main",
		"2 | import \"missing.tx\"",
		"  | ^",
		"4 | func main() int {",
	}
	for _, part := range wantParts {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("error = %q, want it to contain %q", err.Error(), part)
		}
	}
}

func TestRunFileReturnsImportedTypeErrorWithImportedSourceContext(t *testing.T) {
	dir := t.TempDir()
	mainPath := writeTempFile(t, dir, "main.tx", `package main
import "lib.tx"

func main() int {
    return lib.bad()
}`)
	libPath := writeTempFile(t, dir, "lib.tx", `package lib
pub func bad() int {
    return "wrong"
}`)

	var out bytes.Buffer
	err := runFile(&out, mainPath)
	if err == nil {
		t.Fatal("expected error")
	}
	if out.String() != "" {
		t.Fatalf("output = %q, want empty output", out.String())
	}

	wantParts := []string{
		libPath + ":3:12: cannot return string from function returning int",
		"1 | package lib",
		"2 | pub func bad() int {",
		"3 |     return \"wrong\"",
		"  |            ^",
	}
	for _, part := range wantParts {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("error = %q, want it to contain %q", err.Error(), part)
		}
	}
}

func TestRunFileReturnsImportedParseErrorWithImportedSourceContext(t *testing.T) {
	dir := t.TempDir()
	mainPath := writeTempFile(t, dir, "main.tx", `package main
import "lib.tx"

func main() int {
    return 0
}`)
	libPath := writeTempFile(t, dir, "lib.tx", `package main
pub func bad( int {
    return 0
}`)

	var out bytes.Buffer
	err := runFile(&out, mainPath)
	if err == nil {
		t.Fatal("expected error")
	}
	if out.String() != "" {
		t.Fatalf("output = %q, want empty output", out.String())
	}

	wantParts := []string{
		libPath + ":2:15: expected IDENT, got \"int\"",
		"1 | package main",
		"2 | pub func bad( int {",
		"  |               ^",
		"3 |     return 0",
	}
	for _, part := range wantParts {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("error = %q, want it to contain %q", err.Error(), part)
		}
	}
}

func TestRunFileWithDebugWritesPhaseFiles(t *testing.T) {
	requireCC(t)

	t.Cleanup(func() {
		_ = os.RemoveAll("tmp")
	})

	path := writeTempSource(t, `package main
func main() int {
    let x int = 1
    print(x)
    return 0
}`)

	var out bytes.Buffer
	err := runFileWithOptions(&out, path, runOptions{Debug: true})
	if err != nil {
		t.Fatal(err)
	}

	debugDir := filepath.Join("tmp", "trux-debug", "main")
	if !strings.Contains(out.String(), "debug files: "+debugDir) {
		t.Fatalf("output = %q, want debug dir", out.String())
	}

	wantFiles := []string{
		"00-source.tx",
		"01-tokens.txt",
		"02-ast.json",
		"03-types.txt",
		"04-ir.json",
		"05-c.c",
	}
	for _, file := range wantFiles {
		path := filepath.Join(debugDir, file)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected debug file %s: %v", path, err)
		}
	}

	tokens := readFile(t, filepath.Join(debugDir, "01-tokens.txt"))
	if !strings.Contains(tokens, "IDENT\t\"print\"") {
		t.Fatalf("tokens = %q, want print token", tokens)
	}

	typeInfo := readFile(t, filepath.Join(debugDir, "03-types.txt"))
	if !strings.Contains(typeInfo, "print -> print(int)") {
		t.Fatalf("type info = %q, want resolved print call", typeInfo)
	}

	typedIR := readFile(t, filepath.Join(debugDir, "04-ir.json"))
	if !strings.Contains(typedIR, `"Type": "int"`) {
		t.Fatalf("IR = %q, want int print type", typedIR)
	}

	cSource := readFile(t, filepath.Join(debugDir, "05-c.c"))
	if !strings.Contains(cSource, "rt_print_int(trux_v_1_x);") || !strings.Contains(cSource, "rt_print_newline();") {
		t.Fatalf("generated C = %q, want int print call with newline", cSource)
	}
}

func TestRunFileWithDebugWritesPartialFilesOnTypeError(t *testing.T) {
	t.Cleanup(func() {
		_ = os.RemoveAll("tmp")
	})

	path := writeTempSource(t, `package main
func main() int {
    print()
    return 0
}`)

	var out bytes.Buffer
	err := runFileWithOptions(&out, path, runOptions{Debug: true})
	if err == nil {
		t.Fatal("expected error")
	}

	debugDir := filepath.Join("tmp", "trux-debug", "main")
	if !strings.Contains(out.String(), "debug files: "+debugDir) {
		t.Fatalf("output = %q, want debug dir", out.String())
	}
	for _, file := range []string{"00-source.tx", "01-tokens.txt", "02-ast.json"} {
		path := filepath.Join(debugDir, file)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected debug file %s: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(debugDir, "03-types.txt")); !os.IsNotExist(err) {
		t.Fatalf("types debug file should not exist after type error, stat err = %v", err)
	}
}

func TestRunFileReturnsLexError(t *testing.T) {
	path := writeTempSource(t, "@\n")

	var out bytes.Buffer
	err := runFile(&out, path)
	if err == nil {
		t.Fatal("expected error")
	}

	if out.String() != "" {
		t.Fatalf("output = %q, want empty output", out.String())
	}

	wantParts := []string{
		path + ":1:1: lex error: illegal character \"@\"",
		"1 | @",
		"  | ^",
	}
	for _, part := range wantParts {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("error = %q, want it to contain %q", err.Error(), part)
		}
	}
}

func TestRunFileReturnsSyntaxErrorWithSourceContext(t *testing.T) {
	src := `package main

func add(a int, b int) int {
    return a + b
}

func main( int {
    return 0
}`
	path := writeTempSource(t, src)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err == nil {
		t.Fatal("expected error")
	}
	if out.String() != "" {
		t.Fatalf("output = %q, want empty output", out.String())
	}

	wantParts := []string{
		path + ":7:12: expected IDENT, got \"int\"",
		"5 | }",
		"6 | ",
		"7 | func main( int {",
		"  |            ^",
		"8 |     return 0",
		"9 | }",
	}
	for _, part := range wantParts {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("error = %q, want it to contain %q", err.Error(), part)
		}
	}
}

func TestRunFileReturnsTypeErrorWithSourceContext(t *testing.T) {
	src := `package main

func main() int {
    print()
    return 0
}`
	path := writeTempSource(t, src)

	var out bytes.Buffer
	err := runFile(&out, path)
	if err == nil {
		t.Fatal("expected error")
	}
	if out.String() != "" {
		t.Fatalf("output = %q, want empty output", out.String())
	}

	wantParts := []string{
		path + ":4:5: print expects at least 1 argument, got 0",
		"2 | ",
		"3 | func main() int {",
		"4 |     print()",
		"  |     ^",
		"5 |     return 0",
		"6 | }",
	}
	for _, part := range wantParts {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("error = %q, want it to contain %q", err.Error(), part)
		}
	}
}

func TestRunCommandDoesNotPrintUsageForParseErrors(t *testing.T) {
	if !rootCmd.SilenceErrors {
		t.Fatal("root command should let execute print errors once")
	}
	if !runCmd.SilenceUsage {
		t.Fatal("run command should not print usage for source parse errors")
	}
}

func TestEmitCFileWritesGeneratedC(t *testing.T) {
	path := writeTempSource(t, `package main
func main() int {
    return 0
}`)

	result, err := compileFile(path, compileOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result.CSource, "int main(void)") {
		t.Fatalf("generated C = %q, want C main", result.CSource)
	}
}

func TestEmitCCommandWritesSourceAndRuntimeHeader(t *testing.T) {
	path := writeTempSource(t, `package main
func main() int {
    return 0
}`)
	outDir := t.TempDir()
	var out bytes.Buffer

	oldOutDir := emitCOutDir
	t.Cleanup(func() {
		emitCOutDir = oldOutDir
		emitCCmd.SetOut(nil)
	})
	emitCOutDir = outDir
	emitCCmd.SetOut(&out)

	if err := emitCCmd.RunE(emitCCmd, []string{path}); err != nil {
		t.Fatal(err)
	}

	sourcePath := filepath.Join(outDir, filepath.Base(strings.TrimSuffix(path, filepath.Ext(path)))+".c")
	headerPath := filepath.Join(outDir, "trux_runtime.h")
	if _, err := os.Stat(sourcePath); err != nil {
		t.Fatalf("expected generated source: %v", err)
	}
	if _, err := os.Stat(headerPath); err != nil {
		t.Fatalf("expected runtime header: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, sourcePath) || !strings.Contains(got, headerPath) {
		t.Fatalf("output = %q, want written source and header paths", got)
	}
}

func TestEmitCCommandWritesHTTPRuntimeHeader(t *testing.T) {
	path := writeTempSource(t, `package main
import "http"

func handle(request int) int {
    http.respond(request, 200, "text/plain", "ok")
    return 0
}

func main() int {
    http.serve("127.0.0.1", 8080, 2, handle)
    return 0
}`)
	outDir := t.TempDir()
	var out bytes.Buffer

	oldOutDir := emitCOutDir
	t.Cleanup(func() {
		emitCOutDir = oldOutDir
		emitCCmd.SetOut(nil)
	})
	emitCOutDir = outDir
	emitCCmd.SetOut(&out)

	if err := emitCCmd.RunE(emitCCmd, []string{path}); err != nil {
		t.Fatal(err)
	}

	httpHeaderPath := filepath.Join(outDir, "trux_http_runtime.h")
	if _, err := os.Stat(httpHeaderPath); err != nil {
		t.Fatalf("expected HTTP runtime header: %v", err)
	}
	if !strings.Contains(out.String(), httpHeaderPath) {
		t.Fatalf("output = %q, want HTTP runtime header path", out.String())
	}
}

func TestEmitCFileSupportsModuleProgram(t *testing.T) {
	dir := t.TempDir()
	mainPath := writeTempFile(t, dir, "main.tx", `package main
import "math.tx"

func main() int {
    return math.add(1, 2)
}`)
	writeTempFile(t, dir, "math.tx", `package math
pub func add(a int, b int) int {
    return a + b
}`)

	result, err := compileFile(mainPath, compileOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result.CSource, "int64_t trux_f_16___trux_mod_1_add(") {
		t.Fatalf("generated C = %q, want module function add", result.CSource)
	}
	if !strings.Contains(result.CSource, "int main(void)") {
		t.Fatalf("generated C = %q, want C main", result.CSource)
	}
}

func TestBuildCommandRequiresOutput(t *testing.T) {
	oldOutput := buildOutput
	t.Cleanup(func() {
		buildOutput = oldOutput
	})
	buildOutput = ""

	err := buildCmd.RunE(buildCmd, []string{"main.tx"})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "build requires -o <output>" {
		t.Fatalf("error = %q, want output requirement", err.Error())
	}
}

func TestBuildFileRespectsCCEnvironmentVariable(t *testing.T) {
	path := writeTempSource(t, `package main
func main() int {
    return 0
}`)
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "program")
	logPath := filepath.Join(dir, "cc.log")
	fakeCC := writeFakeCompiler(t, filepath.Join(dir, "fake-cc"), logPath)
	t.Setenv("CC", fakeCC)

	if err := buildFile(path, outputPath); err != nil {
		t.Fatal(err)
	}

	log := readFile(t, logPath)
	if !strings.Contains(log, "-std=c11") {
		t.Fatalf("compiler args = %q, want C11 standard flag", log)
	}
	if !strings.Contains(log, "-O2") {
		t.Fatalf("compiler args = %q, want optimized build flag", log)
	}
	if !strings.Contains(log, "-o "+outputPath) {
		t.Fatalf("compiler args = %q, want output path", log)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected output executable: %v", err)
	}
}

func TestCompileGeneratedUsesO0ForDebugGeneratedC(t *testing.T) {
	path := writeTempSource(t, `package main
func main() int {
    return 0
}`)
	result, err := compileFile(path, compileOptions{Debug: true})
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	outputPath := filepath.Join(dir, "program")
	logPath := filepath.Join(dir, "cc.log")
	fakeCC := writeFakeCompiler(t, filepath.Join(dir, "fake-cc"), logPath)
	t.Setenv("CC", fakeCC)
	cPath, err := writeGeneratedFiles(dir, "main", result)
	if err != nil {
		t.Fatal(err)
	}

	if err := compileGenerated(cPath, outputPath, generatedCompileOptions{Debug: result.Debug}); err != nil {
		t.Fatal(err)
	}

	log := readFile(t, logPath)
	if !strings.Contains(log, "-O0") {
		t.Fatalf("compiler args = %q, want debug build flag", log)
	}
	if strings.Contains(log, "-O2") {
		t.Fatalf("compiler args = %q, did not expect optimized debug build", log)
	}
}

func TestCompileGeneratedUsesPthreadForHTTP(t *testing.T) {
	path := writeTempSource(t, `package main
import "http"

func handle(request int) int {
    http.respond(request, 200, "text/plain", "ok")
    return 0
}

func main() int {
    http.serve("127.0.0.1", 8080, 2, handle)
    return 0
}`)
	result, err := compileFile(path, compileOptions{})
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	outputPath := filepath.Join(dir, "program")
	logPath := filepath.Join(dir, "cc.log")
	fakeCC := writeFakeCompiler(t, filepath.Join(dir, "fake-cc"), logPath)
	t.Setenv("CC", fakeCC)
	cPath, err := writeGeneratedFiles(dir, "main", result)
	if err != nil {
		t.Fatal(err)
	}

	if err := compileGenerated(cPath, outputPath, generatedCompileOptions{UsesHTTP: result.UsesHTTP}); err != nil {
		t.Fatal(err)
	}

	log := readFile(t, logPath)
	if !strings.Contains(log, "-pthread") {
		t.Fatalf("compiler args = %q, want -pthread", log)
	}
}

func TestCompileCDetectsHTTPRuntimeInclude(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "main.c")
	outputPath := filepath.Join(dir, "program")
	logPath := filepath.Join(dir, "cc.log")
	fakeCC := writeFakeCompiler(t, filepath.Join(dir, "fake-cc"), logPath)
	t.Setenv("CC", fakeCC)
	if err := os.WriteFile(sourcePath, []byte(`#include "trux_runtime.h"
#include "trux_http_runtime.h"
int main(void) { return 0; }
`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := compileC(sourcePath, outputPath); err != nil {
		t.Fatal(err)
	}

	log := readFile(t, logPath)
	if !strings.Contains(log, "-pthread") {
		t.Fatalf("compiler args = %q, want -pthread", log)
	}
}

func TestGeneratedExamplesCompileWithStrictCWarnings(t *testing.T) {
	requireCC(t)

	paths := exampleEntrypoints(t)
	if len(paths) == 0 {
		t.Fatal("no example programs found")
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			result, err := compileFile(path, compileOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if result.UsesCUDA {
				t.Skip("CUDA examples are compiled by nvcc, not strict C")
			}

			dir := t.TempDir()
			cPath, err := writeGeneratedFiles(dir, "main", result)
			if err != nil {
				t.Fatal(err)
			}

			compiler := os.Getenv("CC")
			if compiler == "" {
				compiler = "cc"
			}
			objPath := filepath.Join(dir, "main.o")
			cmd := exec.Command(compiler, "-std=c11", "-Wall", "-Wextra", "-pedantic", "-c", cPath, "-o", objPath)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("compile generated C: %v\n%s", err, output)
			}
			if strings.TrimSpace(string(output)) != "" {
				t.Fatalf("strict C compiler emitted warnings:\n%s", output)
			}
		})
	}
}

func TestRunFileCompilesAndExecutesGPUMatrixExample(t *testing.T) {
	requireNVCCAndCUDA(t)

	var out bytes.Buffer
	err := runFile(&out, "../../examples/gpu_matrix_multiply.tx")
	if err != nil {
		t.Fatal(err)
	}

	cpuChecksum := outputValue(t, out.String(), "cpu checksum ")
	gpuChecksum := outputValue(t, out.String(), "gpu checksum ")
	if cpuChecksum == "0" {
		t.Fatalf("cpu checksum = 0, want non-zero matrix result")
	}
	if cpuChecksum != gpuChecksum {
		t.Fatalf("checksums differ: cpu=%s gpu=%s\n%s", cpuChecksum, gpuChecksum, out.String())
	}

	wantParts := []string{
		"mismatches 0\n",
		"cpu seconds ",
		"gpu seconds ",
	}
	for _, part := range wantParts {
		if !strings.Contains(out.String(), part) {
			t.Fatalf("output = %q, want it to contain %q", out.String(), part)
		}
	}
}

func outputValue(t *testing.T, output string, prefix string) string {
	t.Helper()

	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix)
		}
	}
	t.Fatalf("output = %q, want line with prefix %q", output, prefix)
	return ""
}

func TestBuildFileRespectsNVCCEnvironmentVariableForGPUProgram(t *testing.T) {
	path := writeTempSource(t, `package main
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
    gpu.free(out)
    return 0
}`)
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "program")
	logPath := filepath.Join(dir, "nvcc.log")
	fakeNVCC := writeFakeCompiler(t, filepath.Join(dir, "fake-nvcc"), logPath)
	t.Setenv("NVCC", fakeNVCC)

	if err := buildFile(path, outputPath); err != nil {
		t.Fatal(err)
	}

	log := readFile(t, logPath)
	if !strings.Contains(log, "-x cu") {
		t.Fatalf("compiler args = %q, want CUDA source flag", log)
	}
	if strings.Contains(log, "-std=c11") {
		t.Fatalf("compiler args = %q, did not expect C11 flag for nvcc", log)
	}
	if !strings.Contains(log, "-o "+outputPath) {
		t.Fatalf("compiler args = %q, want output path", log)
	}
}

func exampleEntrypoints(t *testing.T) []string {
	t.Helper()

	paths, err := filepath.Glob("../../examples/*.tx")
	if err != nil {
		t.Fatal(err)
	}

	err = filepath.WalkDir("../../examples", func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Base(path) == "main.tx" && filepath.Dir(path) != "../../examples" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	return paths
}

func writeTempSource(t *testing.T, src string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "main.tx")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	return path
}

func writeTempFile(t *testing.T, dir string, name string, src string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(content)
}

func freeTCPPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address = %T, want *net.TCPAddr", listener.Addr())
	}
	return addr.Port
}

func requireCC(t *testing.T) {
	t.Helper()

	compiler := os.Getenv("CC")
	if compiler == "" {
		compiler = "cc"
	}
	if _, err := exec.LookPath(compiler); err != nil {
		t.Skipf("C compiler %q not found", compiler)
	}
}

func requireNVCCAndCUDA(t *testing.T) {
	t.Helper()

	compiler := os.Getenv("NVCC")
	if compiler == "" {
		compiler = "nvcc"
	}
	if _, err := exec.LookPath(compiler); err != nil {
		t.Skipf("CUDA compiler %q not found", compiler)
	}
	if _, err := exec.LookPath("nvidia-smi"); err != nil {
		t.Skip("nvidia-smi not found")
	}
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("cl"); err != nil {
			t.Skip("nvcc host compiler cl.exe not found")
		}
	}
	cmd := exec.Command("nvidia-smi")
	if err := cmd.Run(); err != nil {
		t.Skipf("CUDA device unavailable: %v", err)
	}
}

func writeFakeCompiler(t *testing.T, path string, logPath string) string {
	t.Helper()

	if runtime.GOOS == "windows" {
		path += ".cmd"
		script := fmt.Sprintf("@echo off\r\nsetlocal\r\necho %%* > \"%s\"\r\n:loop\r\nif \"%%~1\"==\"\" exit /b 2\r\nif \"%%~1\"==\"-o\" goto writeout\r\nshift\r\ngoto loop\r\n:writeout\r\nshift\r\ntype nul > \"%%~1\"\r\nexit /b 0\r\n", logPath)
		if err := os.WriteFile(path, []byte(script), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	script := fmt.Sprintf(`#!/bin/sh
echo "$@" > %q
while [ "$#" -gt 0 ]; do
    if [ "$1" = "-o" ]; then
        shift
        touch "$1"
        exit 0
    fi
    shift
done
exit 2
`, logPath)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func truxString(value string) string {
	return strconv.Quote(value)
}
