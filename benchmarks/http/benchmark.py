#!/usr/bin/env python3
import subprocess
import time
import re
import json
import os
import sys

PROFILES = [
    {
        "name": "Micro Edge (1 vCPU, 256MB)",
        "cpuset": "0",
        "cpus": "1.0",
        "memory": "256m",
        "workers": 1,
    },
    {
        "name": "Cloud Standard (2 vCPUs, 1GB)",
        "cpuset": "0,1",
        "cpus": "2.0",
        "memory": "1g",
        "workers": 2,
    },
    {
        "name": "Compute Node (4 vCPUs, 2GB)",
        "cpuset": "0,1,2,3",
        "cpus": "4.0",
        "memory": "2g",
        "workers": 4,
    },
]

ENDPOINTS = [
    {
        "name": "Plaintext Baseline (/plaintext)",
        "path": "/plaintext",
        "headers": [],
        "description": "Raw HTTP/1.1 parsing, string constant response, zero dynamic routing overhead",
    },
    {
        "name": "JSON Serialization (/json/serialize)",
        "path": "/json/serialize",
        "headers": [],
        "description": "yyjson document creation, object mutation with typed fields, serialization to JSON string, and HTTP response",
    },
    {
        "name": "SQLite File Insert & Query (/sqlite/insert-and-query)",
        "path": "/sqlite/insert-and-query",
        "headers": [],
        "description": "On-disk SQLite WAL database file (/tmp/bench.db) log row insertion, SELECT query retrieval, and JSON serialization",
    },
    {
        "name": "SQLite File Prepared Query (/sqlite/query-by-id/1024)",
        "path": "/sqlite/query-by-id/1024",
        "headers": [],
        "description": "Path parameter extraction (:id), on-disk SQLite WAL prepared statement, parameter binding, column extraction, and JSON response",
    },
    {
        "name": "Middleware & SQLite File Audit (/api/authenticated-audit)",
        "path": "/api/authenticated-audit",
        "headers": ["-H", "Authorization: Bearer secret-benchmark-token"],
        "description": "Prefix middleware (useAt), Bearer auth validation, on-disk SQLite WAL audit logging insert, context propagation, and privileged data return",
    },
]

CONCURRENCY_LEVELS = [100, 1000]
DURATION = "6s"

def run_cmd(cmd, check=True):
    if isinstance(cmd, list):
        res = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    else:
        res = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, shell=True)
    if check and res.returncode != 0:
        raise RuntimeError(f"Command failed ({res.returncode}): {cmd}\nStdout: {res.stdout}\nStderr: {res.stderr}")
    return res.stdout.strip()

def parse_wrk_output(output):
    req_sec = 0.0
    transfer_sec = "0"
    p50, p75, p90, p99 = "N/A", "N/A", "N/A", "N/A"
    avg_latency = "N/A"

    m_req = re.search(r"Requests/sec:\s+([\d\.]+)", output)
    if m_req:
        req_sec = float(m_req.group(1))

    m_trans = re.search(r"Transfer/sec:\s+([\d\.]+[A-Za-z]+)", output)
    if m_trans:
        transfer_sec = m_trans.group(1)

    m_avg = re.search(r"Latency\s+([\d\.]+[a-z]+)", output)
    if m_avg:
        avg_latency = m_avg.group(1)

    m_50 = re.search(r"50%\s+([\d\.]+[a-z]+)", output)
    if m_50: p50 = m_50.group(1)
    m_75 = re.search(r"75%\s+([\d\.]+[a-z]+)", output)
    if m_75: p75 = m_75.group(1)
    m_90 = re.search(r"90%\s+([\d\.]+[a-z]+)", output)
    if m_90: p90 = m_90.group(1)
    m_99 = re.search(r"99%\s+([\d\.]+[a-z]+)", output)
    if m_99: p99 = m_99.group(1)

    return {
        "req_sec": req_sec,
        "transfer_sec": transfer_sec,
        "avg_latency": avg_latency,
        "p50": p50,
        "p75": p75,
        "p90": p90,
        "p99": p99,
    }

def get_container_mem(container_name):
    try:
        out = run_cmd(f"docker stats --no-stream --format '{{{{.MemUsage}}}}' {container_name}")
        return out.split("/")[0].strip()
    except Exception:
        return "N/A"

def main():
    base_dir = os.path.dirname(os.path.abspath(__file__))
    project_root = os.path.abspath(os.path.join(base_dir, "..", ".."))

    print("=== 1. Compiling Trux HTTP Server Binary ===")
    trux_bin = os.path.join(project_root, "bin", "trux")
    server_tx = os.path.join(base_dir, "server.tx")
    server_bin = os.path.join(base_dir, "server")
    run_cmd(f"cd {project_root} && go build -o {trux_bin} ./cmd/trux")
    run_cmd(f"{trux_bin} build -o {server_bin} {server_tx}")
    print(f"Compiled binary: {server_bin}")

    print("\n=== 2. Building Docker Benchmark Image ===")
    run_cmd(f"docker build -t trux-http-bench {base_dir}")
    print("Docker image 'trux-http-bench' ready.")

    results = []

    print("\n=== 3. Executing Benchmark Matrix ===")
    container_name = "trux-bench-runner"

    for profile in PROFILES:
        print(f"\n=======================================================")
        print(f"  Profile: {profile['name']}")
        print(f"  Limits: cpus={profile['cpus']}, cpuset={profile['cpuset']}, mem={profile['memory']}")
        print(f"=======================================================")

        # Run container
        run_cmd(f"docker rm -f {container_name} 2>/dev/null || true", check=False)
        docker_run_cmd = (
            f"docker run -d --name {container_name} "
            f"--security-opt seccomp=unconfined --ulimit memlock=-1:-1 "
            f"-p 8080:8080 "
            f"--cpuset-cpus='{profile['cpuset']}' --cpus='{profile['cpus']}' "
            f"--memory='{profile['memory']}' "
            f"trux-http-bench"
        )
        run_cmd(docker_run_cmd)
        time.sleep(1.5)

        # Warmup
        run_cmd("curl -s http://127.0.0.1:8080/plaintext > /dev/null")

        for endpoint in ENDPOINTS:
            for c in CONCURRENCY_LEVELS:
                threads = min(c, 8)
                wrk_cmd = [
                    "wrk",
                    f"-t{threads}",
                    f"-c{c}",
                    f"-d{DURATION}",
                    "--latency",
                ] + endpoint["headers"] + [f"http://127.0.0.1:8080{endpoint['path']}"]

                print(f"  -> Testing {endpoint['name']} @ c={c} (threads={threads})...", end="", flush=True)
                raw_wrk = run_cmd(wrk_cmd)
                metrics = parse_wrk_output(raw_wrk)
                mem = get_container_mem(container_name)
                metrics["memory"] = mem

                entry = {
                    "profile": profile["name"],
                    "workers": profile["workers"],
                    "endpoint": endpoint["name"],
                    "path": endpoint["path"],
                    "concurrency": c,
                    **metrics,
                }
                results.append(entry)
                print(f"  {metrics['req_sec']:,.0f} req/s | p50: {metrics['p50']} | p99: {metrics['p99']} | mem: {mem}")

        run_cmd(f"docker rm -f {container_name}")

    # Save results to json
    results_path = os.path.join(base_dir, "results.json")
    with open(results_path, "w") as f:
        json.dump(results, f, indent=2)
    print(f"\nRaw results saved to {results_path}")

    # Generate Markdown Summary
    md_path = os.path.join(base_dir, "BENCHMARK_RESULTS.md")
    with open(md_path, "w") as f:
        f.write("# Trux HTTP io_uring Benchmark Results\n\n")
        f.write(f"*Kernel: {run_cmd('uname -r')}*\n")
        f.write(f"*Docker Engine: {run_cmd('docker version --format \"{{.Server.Version}}\"')}*\n")
        f.write(f"*Duration per test: {DURATION}*\n\n")

        for profile in PROFILES:
            f.write(f"### {profile['name']}\n\n")
            f.write("| Endpoint | Concurrency | Throughput (req/s) | Avg Latency | p50 | p90 | p99 | Memory |\n")
            f.write("| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |\n")
            prof_results = [r for r in results if r["profile"] == profile["name"]]
            for r in prof_results:
                f.write(f"| {r['endpoint']} | {r['concurrency']} | **{r['req_sec']:,.0f}** | {r['avg_latency']} | {r['p50']} | {r['p90']} | {r['p99']} | {r['memory']} |\n")
            f.write("\n")

    print(f"Markdown report generated at {md_path}")

if __name__ == "__main__":
    main()
