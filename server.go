package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// AllowedOrigin specifies permitted origins for CORS. Use "*" for development or set your domain for production.
const AllowedOrigin = "*"

// getClientIP extracts the clean client IP address, handling optional ports and reverse proxies safely
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", AllowedOrigin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Range")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
}

func main() {
	// Ping endpoint
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		ip := getClientIP(r)
		fmt.Printf("[%s] Ping request from %s\n", time.Now().Format("15:04:05"), ip)

		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "%d", time.Now().UnixMilli())
	})

	// Download endpoint
	http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		ip := getClientIP(r)
		fmt.Printf("[%s] Download started by %s\n", time.Now().Format("15:04:05"), ip)

		w.Header().Set("Content-Type", "application/octet-stream")
		// Stream 50 MB using a single 1 MB reusable buffer to keep memory minimal
		chunk := make([]byte, 1024*1024)
		totalMB := 50
		for i := 0; i < totalMB; i++ {
			if _, err := w.Write(chunk); err != nil {
				break
			}
		}
	})

	// Upload endpoint
	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		ip := getClientIP(r)
		fmt.Printf("[%s] Upload started by %s\n", time.Now().Format("15:04:05"), ip)

		// Hard limit upload body size to 100 MB max to prevent memory & connection exhaustion
		r.Body = http.MaxBytesReader(w, r.Body, 100*1024*1024)
		bytesRead, err := io.Copy(io.Discard, r.Body)
		if err != nil {
			fmt.Printf("[%s] Upload error from %s: %v\n", time.Now().Format("15:04:05"), ip, err)
			http.Error(w, "Payload Too Large or request aborted", http.StatusRequestEntityTooLarge)
			return
		}

		fmt.Printf("[%s] Upload completed by %s (received %.2f MB)\n", time.Now().Format("15:04:05"), ip, float64(bytesRead)/(1024*1024))
		fmt.Fprintf(w, "ok")
	})

	// Serve frontend LAST — must be after all other routes
	fs := http.FileServer(http.Dir("frontend"))
	http.Handle("/", fs)

	// Configure HTTP Server with strict execution timeouts against Slowloris attacks
	srv := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Println("Server running on :8080 (with security hardening enabled)")
	err := srv.ListenAndServe()
	if err != nil {
		log.Fatalf("Server error: %v", err)
	}
}