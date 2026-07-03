package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// getClientIP extracts the clean client IP address, handling optional ports and reverse proxies
func getClientIP(r *http.Request) string {
	// Check if request is forwarded by a proxy (like Nginx/Cloudflare)
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	// Fall back to direct TCP address, stripping the port
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func main() {
	// Ping endpoint
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		fmt.Printf("[%s] Ping request from %s\n", time.Now().Format("15:04:05"), ip)

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		fmt.Fprintf(w, "%d", time.Now().UnixMilli())
	})

	// Download endpoint
	http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		fmt.Printf("[%s] Download started by %s\n", time.Now().Format("15:04:05"), ip)

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		data := make([]byte, 10*1024*1024)
		w.Write(data)
	})

	// Upload endpoint
	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		fmt.Printf("[%s] Upload started by %s\n", time.Now().Format("15:04:05"), ip)

		w.Header().Set("Access-Control-Allow-Origin", "*")
		// Read and discard body to prevent connection reset by peer
		bytesRead, _ := io.Copy(io.Discard, r.Body)
		fmt.Printf("[%s] Upload completed by %s (received %.2f MB)\n", time.Now().Format("15:04:05"), ip, float64(bytesRead)/(1024*1024))
		fmt.Fprintf(w, "ok")
	})

	// Serve frontend LAST — must be after all other routes
	fs := http.FileServer(http.Dir("frontend"))
	http.Handle("/", fs)

	fmt.Println("Server running on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error:", err)
	}
}