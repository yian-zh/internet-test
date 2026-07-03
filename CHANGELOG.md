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
