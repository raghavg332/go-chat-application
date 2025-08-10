# Go Multi-Goroutine Chat Application

A high-performance, terminal-based chat system written in Go with **TCP sockets** and **goroutines**.  
Supports **multi-client messaging**, **group chat**, and a simple command interface — engineered for concurrency, stability, and real-time feedback.

---

## System Architecture
<img width="1112" height="605" alt="system-architecture" src="https://github.com/user-attachments/assets/8a6fdb9b-6441-4ecd-ab7f-b57e9e09859c" />

---

## Message Flow
<img width="1490" height="800" alt="message flow" src="https://github.com/user-attachments/assets/a027c1d5-89e4-4837-ae88-fa85cd20cbf6" />

---

## Features

### 🖥 Server
- **Concurrent TCP server** with one **goroutine per client** (Go’s M:N scheduler).
- **Group chat support** (`/join <group>`, `/leave`, `/groups`).
- **Global chat** broadcasting to all connected users.
- **User list** (`/users`) in real time.
- **Thread-safe state management** with `sync.Mutex` to prevent race conditions.
- **Graceful disconnect handling** with Ctrl+C cleanup.

### 💬 Client
- **Terminal UI** over stdin/stdout with ANSI escape sequences for clean display.
- **Color-ready output** (easy to extend with ANSI color for user/prefix differentiation).
- **Responsive input** box with prompt.
- **Ctrl+C safe exit** — cleans up sockets before exiting.

> For a full TUI, you can swap in a Go TUI library like `tcell` or `bubbletea` without changing the protocol.

---

## 🛠 Tech Stack
- **Language:** Go 1.21+
- **Stdlib:** `net`, `sync`, `os/signal`, `syscall`
- **Protocol:** TCP (IPv4)
- **Platform:** POSIX systems (Linux, macOS)
- **Build/Run:** `go build`, `go run`, optional `-race` for race detection

---

## 📂 Project Structure
```
├── server/
│   └── main.go     # Concurrent chat server (goroutine-per-connection)
├── client/
│   └── client.go   # Terminal chat client
├── testing/
│   └── load_tester.py  # Async load tester for performance benchmarking
└── README.md
```

---

## 🚀 Getting Started

### 1) Build
```bash
# From repo root
go build -o bin/server ./server
go build -o bin/client ./client
```

### 2) Run the server
```bash
./bin/server
# or
go run ./server
```
By default, the server listens on **port 8080**.
Note: An instance of the server is already hosted at 13.200.235.191:8080 that the client can easily connect to.

### 3) Run the client
```bash
./bin/client
# or
go run ./client
```
When prompted, enter a username, then chat using:
- `/users` — List connected users
- `/join <group>` — Create/join a group
- `/groups` — List available groups
- `/leave` — Leave current group

---

## 📊 Performance Benchmark

Use the same `testing/load_tester.py` harness as the C++ version to measure Time-to-First-Byte (TTFB) for `/users` command responses.

**Example baseline (from earlier C++ benchmark on t2.micro):**

| Metric                   | Value     |
|--------------------------|-----------|
| Connections OK           | 200       |
| Connections Failed       | 0         |
| Requests Sent            | 1,500     |
| Responses Observed       | 1,500     |
| **p50 TTFB**             | 53.27 ms  |
| **p90 TTFB**             | 76.43 ms  |
| **p95 TTFB**             | 83.28 ms  |
| **p99 TTFB**             | 94.37 ms  |
| Bytes Sent               | 10,180    |
| Bytes Received           | 2,253,882 |
| Duration                 | 30.02 s   |

### Scalability Sweep
| Clients | p50 TTFB (ms) | p90 TTFB (ms) | Connections OK | Connections Failed |
|---------|---------------|---------------|----------------|--------------------|
| 50      | 34.44         | 99.99         | 100            | 0                  |
| 100     | 55.65         | 84.35         | 200            | 0                  |
| 150     | 72.41         | 198.23        | 300            | 0                  |
| 200     | 119.21        | 207.47        | 400            | 0                  |
| 250     | 84.46         | 173.66        | 500            | 0                  |
| 300     | 143.09        | 246.62        | 600            | 0                  |
| 350     | 150.64        | 450.58        | 700            | 0                  |
| **400** | **165.09**    | **401.11**    | **800**        | **0**              |
| 450     | 233.21        | 832.76        | 900            | 0                  |

> Re-run benchmarks on the Go server to get updated performance metrics.

---

## 📈 Key Takeaways
- **Goroutine-per-client** model = simple code, high concurrency.
- **sync.Mutex** ensures safe shared state for users/groups.
- **Graceful shutdown** cleans up all connections.
- On modest hardware (t2.micro), should scale to **hundreds of concurrent clients**.
