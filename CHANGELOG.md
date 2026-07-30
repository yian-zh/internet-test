# Codebase Modifications & Time Precision Upgrades

This document details the optimizations and changes introduced to improve latency measurements, resource management, error handling, and observability in the speed test application.

---

## 1. High-Precision Timing Switch in `client.py`

### Chrono Resolution Comparison
* **Before**: Used `time.time()`
* **After**: Switched to `time.perf_counter()`

### Rationale
* **Precision**: `time.time()` returns system wall-clock time, which on Windows has a very low resolution (~10–16 ms) and is susceptible to sudden clock jumps (e.g., NTP synchronization or user setting changes).
* **Reliability**: `time.perf_counter()` is a monotonic clock with the highest available resolution on the host system. It cannot go backwards and is specifically designed for benchmarking short intervals, providing highly accurate round-trip times (RTT) and jitter statistics.

---

## 2. Client-Side Enhancements (`client.py`)

### TCP Connection Pooling
* **Before**: Initiated a new TCP handshake on every single request using direct `requests.get()` calls.
* **After**: Implemented `requests.Session()` to reuse a single persistent TCP socket (HTTP Keep-Alive). This isolates the actual network transmission delay from the overhead of repeated TCP 3-way handshakes.

### Socket Warm-up
* **Before**: The first measurement started immediately on a cold connection.
* **After**: Added a warm-up HTTP call before the measurement loop begins, discarding its result to ensure the TCP handshake overhead is not factored into statistics.

### Congestion Mitigation
* **Before**: Looped instantly without sleeping, potentially saturating network interfaces or causing bufferbloat.
* **After**: Added a small delay (`time.sleep(0.05)`) between ping requests to allow the network interface to clear.

### Robustness & Packet Loss
* **Before**: Unhandled network exceptions would crash the entire client script.
* **After**: Wrapped network calls in `try...except requests.RequestException` blocks. If a request drops:
  * The error is caught safely.
  * A value of `None` is appended to the RTT list.
  * Packet loss is computed as a percentage: `loss = (rtts.count(None) / len(rtts)) * 100`.
  * Lost packets are filtered out when calculating `avg`, `min`, `max`, and `jitter` statistics.

### Optimizing Download Overhead
* **Before**: Fetched the complete 10MB test file directly into memory using `r.content`.
* **After**: Switched to streaming (`stream=True`) and read data in `256 KB` blocks (`r.iter_content`). This minimizes local memory allocations and CPU cycles, ensuring local system performance does not bottleneck the network speed calculation.

### Output Formatting & CSV Upgrades
* **Before**: Wrote to `results.csv` without headers and without preventing Windows newline issues.
* **After**:
  * Added auto-header generation (`Target, Avg RTT (ms), Min RTT (ms)...`) for new or empty files.
  * Opened file with `newline=''` to prevent blank line duplication on Windows.
  * Integrated the new **Jitter (ms)** column into logs.

---

## 3. Server-Side Enhancements (`server.go`)

### Clean IP Tracking
* **Before**: Used raw connection addresses (`r.RemoteAddr`), which include port numbers and can be misidentified when behind proxies.
* **After**: Added `getClientIP()` helper to inspect reverse-proxy headers (`X-Forwarded-For`, `X-Real-IP`) and clean TCP ports using `net.SplitHostPort`.

### Server Observability
* **Before**: The Go console application printed nothing during operations.
* **After**: Integrated timestamped terminal logs for all three core endpoints:
  * `/ping`: `[15:04:05] Ping request from 10.0.9.x`
  * `/download`: `[15:04:05] Download started by 10.0.9.x`
  * `/upload`: `[15:04:05] Upload started/completed by 10.0.9.x (received X.XX MB)`

---

## 4. Scaled 50MB Payload & TCP Warmup (`client.py` & `server.go`)

### Test Payload Expansion
* **Before**: Download and upload tests sent 10 MB per stream (40 MB total over 4 parallel streams), completing in ~0.1s over 1 Gbps and causing CPU scheduling variance.
* **After**: Increased test payload size to 50 MB per stream (200 MB total over 4 streams). Measurements now span ~2 seconds, allowing TCP throughput to stabilize at physical line rates (~940–1000+ Mbps).

### Zero-RAM Streaming in `server.go`
* **Before**: Allocated a single large `10MB` slice per request (`make([]byte, 10*1024*1024)`).
* **After**: Streamed 50 MB payloads using a single reusable 1 MB buffer chunk. Server memory consumption remains near 0 MB regardless of request volume.

### TCP Congestion Window Warmup
* **Before**: Speed timers started immediately on cold streams, factoring TCP Slow-Start ramp-up delays into throughput averages.
* **After**: Added pre-test warmup requests to ramp up TCP congestion windows before launching timed parallel streams.

---

## 5. Security & Production Hardening (`server.go`)

### Unbounded Upload Protection
* **Before**: `/upload` used raw `io.Copy(io.Discard, r.Body)`, allowing malicious endless upload streams.
* **After**: Wrapped request bodies with `http.MaxBytesReader(w, r.Body, 100*1024*1024)` to hard-cap uploads at 100 MB max per request with `HTTP 413 Payload Too Large`.

### Connection Timeout Guard (Slowloris Protection)
* **Before**: Used default `http.ListenAndServe(":8080", nil)` with zero timeouts, leaving sockets vulnerable to connection exhaustion.
* **After**: Configured explicit `http.Server` timeouts:
  * `ReadTimeout`: 15 seconds
  * `WriteTimeout`: 30 seconds
  * `IdleTimeout`: 60 seconds

### CORS & Proxy Header Standardization
* **Before**: Static CORS headers per handler and basic IP extraction.
* **After**: Added `enableCORS` helper with `HTTP OPTIONS` preflight support, `no-cache` cache control, and safe extraction of the primary client IP from comma-separated `X-Forwarded-For` proxy chains.
