package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	duration := flag.Duration("duration", 6*time.Second, "length of each concurrency level")
	concurrency := flag.String("concurrency", "1,8,32", "comma-separated keep-alive connection counts")
	modelPath := flag.String("model", "", "TXM1 file to serve; empty writes the synthetic 519-vector model")
	flag.Parse()

	if runtime.GOOS != "linux" {
		return fmt.Errorf("benchmarks/digits needs Linux, liburing, and CUDA")
	}

	levels, err := parseLevels(*concurrency)
	if err != nil {
		return err
	}

	dir := sourceDir()
	root := filepath.Join(dir, "..", "..")
	raw, label, err := loadModelBytes(*modelPath)
	if err != nil {
		return err
	}
	model, err := parseModel(raw)
	if err != nil {
		return err
	}
	want := model.voteDigit(model.scores(zeroImage()))
	body := zeroBody()
	expected := []byte(fmt.Sprintf(`{"digit":%d}`, want))

	txmPath := filepath.Join(dir, "digits.txm")
	if err := os.WriteFile(txmPath, raw, 0o644); err != nil {
		return err
	}
	if *modelPath == "" || !sameFile(*modelPath, txmPath) {
		defer func() { _ = os.Remove(txmPath) }()
	}

	fmt.Fprintf(os.Stderr, "building digits server\n")
	truxBin := filepath.Join(root, "bin", "trux")
	serverBin := filepath.Join(dir, "server")
	serverTx := filepath.Join(dir, "server.tx")
	if err := runCmd(root, "go", "build", "-o", truxBin, "./cmd/trux"); err != nil {
		return err
	}
	if err := runCmd(root, truxBin, "build", "-o", serverBin, serverTx); err != nil {
		return err
	}

	logFile, err := os.CreateTemp("", "trux-digits-bench-*.log")
	if err != nil {
		return err
	}
	logPath := logFile.Name()
	defer func() {
		_ = logFile.Close()
		_ = os.Remove(logPath)
	}()

	cmd := exec.Command(serverBin)
	cmd.Dir = dir
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		return err
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		select {
		case <-exited:
		case <-time.After(2 * time.Second):
		}
	}()

	url := "http://127.0.0.1:8090/predict"
	if err := waitReady(url, body, expected, exited, logPath); err != nil {
		return err
	}

	fmt.Printf("handwritten digits HTTP+GPU\n")
	fmt.Printf("model %s  n_sv %d  features %d  gamma %g\n", label, model.nSV, digitsFeatures, model.gamma)
	fmt.Printf("listen 127.0.0.1:8090  workers one per CPU in the affinity mask\n")
	fmt.Printf("check 64 zeros -> {\"digit\":%d}\n", want)
	fmt.Printf("one launch per request per worker; weights stay mounted\n")

	failed := false
	for _, level := range levels {
		result, err := runLevel(url, body, expected, level, *duration, exited, logPath)
		if err != nil {
			return err
		}
		fmt.Printf("concurrency %d  requests %d  req/s %.1f  p50 %s  p99 %s  errors %d\n",
			level, result.ok, result.perSec, formatMillis(result.p50), formatMillis(result.p99), result.errors)
		if result.errors > 0 || result.ok == 0 {
			failed = true
		}
	}
	if failed {
		return fmt.Errorf("benchmark had errors or no successful requests")
	}
	return nil
}

type levelResult struct {
	ok     int
	errors int
	perSec float64
	p50    time.Duration
	p99    time.Duration
}

func runLevel(url string, body []byte, expected []byte, concurrency int, duration time.Duration, exited <-chan error, logPath string) (levelResult, error) {
	client := newClient(concurrency)
	defer client.CloseIdleConnections()
	started := time.Now()
	deadline := started.Add(duration)
	latencies := make([][]time.Duration, concurrency)
	errors := make([]int, concurrency)
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for time.Now().Before(deadline) {
				t0 := time.Now()
				if err := postEquals(client, url, body, expected); err != nil {
					errors[i]++
					continue
				}
				latencies[i] = append(latencies[i], time.Since(t0))
			}
		}(i)
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case err := <-exited:
		return levelResult{}, fmt.Errorf("server exited: %v\n%s", err, readLog(logPath))
	case <-done:
	}
	elapsed := time.Since(started)
	var all []time.Duration
	var result levelResult
	for i := 0; i < concurrency; i++ {
		all = append(all, latencies[i]...)
		result.ok += len(latencies[i])
		result.errors += errors[i]
	}
	if result.ok > 0 && elapsed > 0 {
		result.perSec = float64(result.ok) / elapsed.Seconds()
		sort.Slice(all, func(i, j int) bool { return all[i] < all[j] })
		result.p50 = percentile(all, 0.50)
		result.p99 = percentile(all, 0.99)
	}
	return result, nil
}

func waitReady(url string, body []byte, expected []byte, exited <-chan error, logPath string) error {
	client := newClient(1)
	defer client.CloseIdleConnections()
	deadline := time.Now().Add(30 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		select {
		case err := <-exited:
			return fmt.Errorf("server exited: %v\n%s", err, readLog(logPath))
		default:
		}
		last = postEquals(client, url, body, expected)
		if last == nil {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("server did not answer: %v\n%s", last, readLog(logPath))
}

func postEquals(client *http.Client, url string, body []byte, expected []byte) error {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK || !bytes.Equal(bytes.TrimSpace(got), expected) {
		return fmt.Errorf("status %d body %q", resp.StatusCode, got)
	}
	return nil
}

func newClient(concurrency int) *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        concurrency,
			MaxIdleConnsPerHost: concurrency,
			MaxConnsPerHost:     concurrency,
			DisableCompression:  true,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}

func zeroBody() []byte {
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i := 0; i < digitsFeatures; i++ {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteByte('0')
	}
	buf.WriteByte(']')
	return buf.Bytes()
}

func parseLevels(text string) ([]int, error) {
	parts := strings.Split(text, ",")
	levels := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 1 {
			return nil, fmt.Errorf("concurrency must be positive integers, got %q", text)
		}
		levels = append(levels, n)
	}
	if len(levels) == 0 {
		return nil, fmt.Errorf("concurrency is empty")
	}
	return levels, nil
}

func loadModelBytes(path string) ([]byte, string, error) {
	if path == "" {
		return syntheticTXM(), "synthetic", nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	return raw, path, nil
}

func percentile(samples []time.Duration, p float64) time.Duration {
	if len(samples) == 0 {
		return 0
	}
	if len(samples) == 1 {
		return samples[0]
	}
	rank := p * float64(len(samples)-1)
	lo := int(rank)
	hi := lo + 1
	if hi >= len(samples) {
		return samples[len(samples)-1]
	}
	frac := rank - float64(lo)
	return time.Duration(float64(samples[lo])*(1-frac) + float64(samples[hi])*frac)
}

func formatMillis(d time.Duration) string {
	return fmt.Sprintf("%.3f ms", float64(d)/float64(time.Millisecond))
}

func sourceDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Dir(file)
}

func sameFile(a string, b string) bool {
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return false
	}
	return aa == bb
}

func runCmd(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w\n%s", name, err, out)
	}
	return nil
}

func readLog(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	if len(data) > 4096 {
		data = data[len(data)-4096:]
	}
	return string(data)
}
