package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	// Ping endpoint
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		fmt.Fprintf(w, "%d", time.Now().UnixMilli())
	})

	// Download endpoint
	http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		data := make([]byte, 10*1024*1024)
		w.Write(data)
	})

	// Upload endpoint
	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		r.Body.Close()
		fmt.Fprintf(w, "ok")
	})

	// Serve frontend LAST — must be after all other routes
	fs := http.FileServer(http.Dir(`C:\Users\ADMIN PC.BENNY\speed-test\frontend`))
http.Handle("/", fs)

	fmt.Println("Server running on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error:", err)
	}
}