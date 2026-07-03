import requests
import time
import csv
import os

TARGET = "http://10.0.9.101:8080"

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
        r = session.get(f"{TARGET}/ping", timeout=1.0)
        r.raise_for_status()
        rtt = (time.perf_counter() - start) * 1000
        rtts.append(rtt)
    except requests.RequestException:
        rtts.append(None)
    
    # Tiny sleep to avoid network/buffer congestion skewing the results
    time.sleep(0.05)

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
    start = time.perf_counter()
    # stream=True allows us to read in larger chunks rather than using requests default small chunk sizes
    r = session.get(f"{TARGET}/download", stream=True, timeout=5.0)
    r.raise_for_status()
    bytes_received = 0
    # Read in 256KB chunks for maximum speed and minimum memory/CPU overhead in Python
    for chunk in r.iter_content(chunk_size=256 * 1024):
        bytes_received += len(chunk)
    elapsed = time.perf_counter() - start
    download_mbps = (bytes_received * 8) / elapsed / 1_000_000
    print(f"Download: {download_mbps:.2f} Mbps")
except requests.RequestException as e:
    print(f"Download speed test failed: {e}")

# --- UPLOAD SPEED TEST ---
upload_mbps = 0.0
try:
    data = b'0' * (10 * 1024 * 1024)  # 10 MB
    start = time.perf_counter()
    r = session.post(f"{TARGET}/upload", data=data, timeout=10.0)
    r.raise_for_status()
    elapsed = time.perf_counter() - start
    upload_mbps = (len(data) * 8) / elapsed / 1_000_000
    print(f"Upload: {upload_mbps:.2f} Mbps")
except requests.RequestException as e:
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