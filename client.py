import requests
import time
import csv
import os
import socket
from concurrent.futures import ThreadPoolExecutor
from urllib3.connection import HTTPConnection

# Disable Nagle's algorithm (TCP_NODELAY) globally for urllib3/requests
HTTPConnection.default_socket_options = (
    HTTPConnection.default_socket_options + [
        (socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
    ]
)

TARGET = "http://localhost:8080"

# --- PING / LATENCY TEST ---
rtts = []
session = requests.Session()

# Warm up the TCP connection so the first measurement doesn't include handshake overhead
try:
    session.get(f"{TARGET}/ping", timeout=2.0)
except requests.RequestException:
    pass

for _ in range(100):
    try:
        start = time.perf_counter()
        r = session.get(f"{TARGET}/ping")
        r.raise_for_status()
        rtt = (time.perf_counter() - start) * 1000
        rtts.append(rtt)
    except requests.RequestException:
        rtts.append(None)
    
    # Tiny sleep to avoid network/buffer congestion skewing the results
    # time.sleep(0.05)

# Calculate statistics ignoring lost packets (None)
valid_rtts = [r for r in rtts if r is not None]
loss = (rtts.count(None) / len(rtts)) * 100

if valid_rtts:
    avg_rtt = sum(valid_rtts) / len(valid_rtts)
    min_rtt = min(valid_rtts)
    max_rtt = max(valid_rtts)
    if len(valid_rtts) > 1:
        jitter = sum(abs(valid_rtts[i] - valid_rtts[i-1]) for i in range(1, len(valid_rtts))) / (len(valid_rtts) - 1)
    else:
        jitter = 0.0
else:
    avg_rtt = 0.0
    min_rtt = 0.0
    max_rtt = 0.0
    jitter = 0.0

print(f"Jitter: {jitter:.2f} ms")
print(f"Avg Delay: {avg_rtt:.2f} ms")
print(f"Min: {min_rtt:.2f} ms | Max: {max_rtt:.2f} ms")
print(f"Packet loss: {loss:.1f}%")

# --- DOWNLOAD SPEED TEST ---
download_mbps = 0.0
try:
    def download_stream():
        s = requests.Session()
        r = s.get(f"{TARGET}/download", stream=True, timeout=5.0)
        r.raise_for_status()
        bytes_received = 0
        for chunk in r.iter_content(chunk_size=256 * 1024):
            bytes_received += len(chunk)
        return bytes_received

    num_threads = 4
    start = time.perf_counter()
    with ThreadPoolExecutor(max_workers=num_threads) as executor:
        futures = [executor.submit(download_stream) for _ in range(num_threads)]
        total_bytes = sum(f.result() for f in futures)
    
    elapsed = time.perf_counter() - start
    download_mbps = (total_bytes * 8) / elapsed / 1_000_000
    print(f"Download: {download_mbps:.2f} Mbps (using {num_threads} parallel streams)")
except Exception as e:
    print(f"Download speed test failed: {e}")

# --- UPLOAD SPEED TEST ---
upload_mbps = 0.0
try:
    data = b'0' * (10 * 1024 * 1024)  # 10 MB per stream
    def upload_stream():
        s = requests.Session()
        r = s.post(f"{TARGET}/upload", data=data, timeout=10.0)
        r.raise_for_status()
        return len(data)

    num_threads = 4
    start = time.perf_counter()
    with ThreadPoolExecutor(max_workers=num_threads) as executor:
        futures = [executor.submit(upload_stream) for _ in range(num_threads)]
        total_bytes = sum(f.result() for f in futures)
        
    elapsed = time.perf_counter() - start
    upload_mbps = (total_bytes * 8) / elapsed / 1_000_000
    print(f"Upload: {upload_mbps:.2f} Mbps (using {num_threads} parallel streams)")
except Exception as e:
    print(f"Upload speed test failed: {e}")

# --- SAVE RESULTS TO CSV ---
file_exists = os.path.exists("results.csv") and os.path.getsize("results.csv") > 0
# Use newline='' on Windows to prevent extra blank rows between writes
with open("results.csv", "a", newline='') as f:
    writer = csv.writer(f)
    if not file_exists:
        writer.writerow(["Target", "Avg RTT (ms)", "Min RTT (ms)", "Max RTT (ms)", "Jitter (ms)", "Download (Mbps)", "Upload (Mbps)"])
    writer.writerow([TARGET, avg_rtt, min_rtt, max_rtt, jitter, download_mbps, upload_mbps])

print("Results saved to results.csv!")