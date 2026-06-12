import requests
import time
import csv

TARGET = "http://localhost:8080"

# --- PING / LATENCY TEST ---c
rtts = []
for _ in range(100):
    start = time.time()
    requests.get(f"{TARGET}/ping")
    rtt = (time.time() - start) * 1000
    rtts.append(rtt)

avg_rtt = sum(rtts) / len(rtts)
min_rtt = min(rtts)
max_rtt = max(rtts)
loss = rtts.count(None) / 100 * 100

# ---jitter formula --- #

jitter = sum(abs(rtts[i] - rtts[i-1]) for i in range(1, len(rtts))) / (len(rtts) - 1)
print(f"Jitter: {jitter:.2f} ms")

print(f"Avg Delay: {avg_rtt:.2f} ms")
print(f"Min: {min_rtt:.2f} ms | Max: {max_rtt:.2f} ms")
print(f"Packet loss: {loss}%")

# --- DOWNLOAD SPEED TEST ---
start = time.time()
r = requests.get(f"{TARGET}/download", stream=True)
bytes_received = len(r.content)
elapsed = time.time() - start
download_mbps = (bytes_received * 8) / elapsed / 1_000_000
print(f"Download: {download_mbps:.2f} Mbps")

# --- UPLOAD SPEED TEST ---
data = b'0' * (10 * 1024 * 1024)  # 10 MB
start = time.time()
requests.post(f"{TARGET}/upload", data=data)
elapsed = time.time() - start
upload_mbps = (len(data) * 8) / elapsed / 1_000_000
print(f"Upload: {upload_mbps:.2f} Mbps")

# --- SAVE RESULTS TO CSV ---
with open("results.csv", "a") as f:
    writer = csv.writer(f)
    writer.writerow([TARGET, avg_rtt, min_rtt, max_rtt, download_mbps, upload_mbps])

print("Results saved to results.csv!")