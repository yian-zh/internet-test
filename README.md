# Internet Performance Benchmarking Engine

An enterprise-grade, lightweight, and high-precision network speed test suite built in Go and Python, featuring an interactive Web interface powered by LibreSpeed. Designed for both LAN (direct Ethernet physical testing) and WAN (public internet deployment) throughput, latency, and jitter measurements.

---

## System Architecture

```text
                  +-----------------------------------+
                  |        Web Browser Frontend       |
                  |     (HTML/CSS/JS - LibreSpeed)    |
                  +-----------------+-----------------+
                                    |
                                    v (HTTP / REST API)
+-------------------+     +---------+---------+     +-------------------+
|  Python Benchmark | <-> |   Go High-Speed   | <-> |   CSV Logger &    |
|   Client Engine   |     |    HTTP Server    |     | Performance Audit |
+-------------------+     +-------------------+     +-------------------+
```

### Architectural Breakdown
1. **High-Speed Go Backend (`server.go`)**:
   - Built on Go's `net/http` runtime for multi-threaded, low-overhead HTTP request processing.
   - Zero-RAM memory allocation via 1 MB buffer chunk streaming for 50 MB payloads.
   - Security-hardened against Denial of Service (DoS), Slowloris connection exhaustion, and header spoofing.
2. **Python Automated Benchmark Client (`client.py`)**:
   - Multi-stream parallel downloader and uploader using `concurrent.futures.ThreadPoolExecutor`.
   - Uses `time.perf_counter()` high-resolution monotonic timers.
   - Applies TCP connection warmup and disables Nagle's algorithm (`TCP_NODELAY`) to isolate physical network throughput.
3. **Web Frontend (`frontend/`)**:
   - Interactive UI utilizing LibreSpeed web worker engine for real-time visual gauges.

---

## Key Features

### Python Benchmark Client (`client.py`) Features

- **Global Nagle's Algorithm Disabling (`TCP_NODELAY`)**:
  - Modifies `urllib3.connection.HTTPConnection.default_socket_options` globally to enable `TCP_NODELAY`.
  - Forces packets to be transmitted immediately without TCP buffering delay, isolating physical network performance.

- **Persistent Socket Pooling & Warmup**:
  - Leverages `requests.Session()` HTTP Keep-Alive sockets across probe loops.
  - Runs pre-benchmark connection warmup calls for ping, download, and upload tests to eliminate TCP 3-way handshake and slow-start overheads from timed windows.

- **Microsecond Monotonic Timing (`time.perf_counter`)**:
  - Uses hardware-backed monotonic counter `time.perf_counter()` for sub-microsecond precision unaffected by system clock shifts or NTP synchronization.

- **Robust 100-Probe Latency & Jitter Analyzer**:
  - Fires 100 sequential latency probes to calculate **Average RTT**, **Minimum RTT**, **Maximum RTT**, **Jitter** (delay variation), and **Packet Loss Percentage**.
  - Safely handles connection drops by filtering `None` values and computing exact packet loss metrics.

- **Multi-Threaded Parallel Download Engine**:
  - Executes 4 parallel download worker streams via `concurrent.futures.ThreadPoolExecutor`.
  - Reads data in optimized `256 KB` stream chunks (`iter_content`) to prevent local RAM and CPU bottlenecking.

- **Multi-Threaded Parallel Upload Engine**:
  - Transmits 50 MB binary payload buffers per thread (200 MB total across 4 streams).
  - Pre-warms upload congestion windows before timing execution over a 15-second socket window.

- **Automated CSV Logging & Persistence (`results.csv`)**:
  - Automatically formats and appends benchmark results (`Target, Avg RTT, Min RTT, Max RTT, Jitter, Download Mbps, Upload Mbps`) into `results.csv`.
  - Features auto-header detection and Windows-safe `newline=''` file handling.

### Backend Server (`server.go`) Features

- **Zero-RAM Chunked Streaming**: Server streams 50 MB payloads using a single reusable 1 MB memory slice, preventing RAM bloat under heavy concurrent loads.
- **Production-Ready Security Hardening**:
  - **100 MB Upload Cap**: `http.MaxBytesReader` blocks infinite stream upload DoS attacks.
  - **Slowloris Timeout Guard**: Strict execution timeouts (`ReadTimeout: 15s`, `WriteTimeout: 30s`, `IdleTimeout: 60s`).
  - **CORS & Preflight Handling**: Configurable `Access-Control-Allow-Origin` and `HTTP OPTIONS` preflight support.
  - **Proxy Header IP Parsing**: Safely extracts client IP from comma-separated `X-Forwarded-For` chains.

---

## Methodologies & Mathematical Formulas

### 1. Round-Trip Time (RTT) & Packet Loss
For 100 sequential probe requests sent to `/ping`:

```text
Packet Loss (%) = ( Failed Probes / Total Probes ) * 100
Average RTT (ms) = ( Sum of all valid RTTs ) / Total Valid Probes
```

### 2. Jitter Calculation
Jitter measures the average delay variation across consecutive ping probes:

```text
Jitter (ms) = Sum( | RTT_(i+1) - RTT_i | ) / ( Total Valid Probes - 1 )
```

### 3. Throughput Calculation (Mbps)
Total bytes transferred over 4 parallel streams divided by elapsed monotonic time (delta t):

```text
Throughput (Mbps) = ( Total Bytes Transferred * 8 ) / ( Elapsed Seconds * 1,000,000 )
```

---

## Technical Stack & Libraries

| Component | Technology / Library | Purpose |
| :--- | :--- | :--- |
| **Backend Language** | Go (1.20+) | High-throughput concurrent HTTP server |
| **Backend HTTP Engine** | `net/http` standard library | Zero-dependency HTTP routing & buffer streaming |
| **Client Language** | Python 3.8+ | CLI benchmark & CSV data logger |
| **HTTP Client Library** | `requests` & `urllib3` | Persistent socket pooling & `TCP_NODELAY` configuration |
| **Concurrency Library** | `concurrent.futures` | Multi-threaded parallel stream execution |
| **Frontend Framework** | Vanilla JS + LibreSpeed | Real-time gauge visualization & web worker execution |

---

## Hardware Setup Guide (PC-to-PC Direct Cable Testing)

For physical hardware benchmarking between two PCs without a router:

1. **Physical Cable**: Connect a Cat5e or Cat6 Ethernet cable between **PC A** (Server) and **PC B** (Client). Modern NICs handle Auto-MDIX automatically.
2. **Static IP Assignment**:
   - **PC A (Server)**: IP `10.0.0.1`, Subnet Mask `255.255.255.0`
   - **PC B (Client)**: IP `10.0.0.2`, Subnet Mask `255.255.255.0`
3. **Firewall & Profile Rules**:
   Set network profile to **Private** and allow port 8080 & ICMP in Windows Firewall (PowerShell as Administrator):
   ```powershell
   Set-NetConnectionProfile -NetworkCategory Private
   netsh advfirewall firewall add rule name="Allow ICMP" protocol=icmpv4:8,any dir=in action=allow
   netsh advfirewall firewall add rule name="SpeedTest 8080" dir=in action=allow protocol=TCP localport=8080
   ```

---

## Known Limitations & Hardware Boundaries

1. **Physical Ethernet Overhead (Theoretical Max)**:
   - On a physical 1 Gbps ($1000 \text{ Mbps}$) link, maximum theoretical TCP payload throughput is $\approx 940 - 960 \text{ Mbps}$ due to Ethernet framing (18–26B), IP headers (20B), and TCP headers (20B).
2. **OS Scheduling Jitter**:
   - Non-realtime desktop operating systems (Windows/Linux) incur $\sim 0.1 - 0.4 \text{ ms}$ scheduling jitter due to CPU context switching and background DPC queues.
3. **Browser Timer Resolution**:
   - Browser security constraints cap `performance.now()` resolution to microsecond buckets to mitigate side-channel timing attacks.

---

## Public Internet Deployment Recommendations

When deploying on public cloud infrastructure (AWS, DigitalOcean, Hetzner):

1. **Reverse Proxy Setup**: Put **Nginx** or **Caddy** in front of port 8080 to handle TLS termination (HTTPS).
2. **DNS & SSL**: Obtain a free Let's Encrypt SSL certificate via `certbot`.
3. **CORS Locking**: Change `const AllowedOrigin = "*"` in `server.go` to your specific domain (`https://speedtest.yourdomain.com`).
4. **Rate Limiting**: Configure Nginx rate limiting (`limit_req_zone`) to restrict requests per IP per minute.
