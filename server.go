package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver driver
)

var db *sql.DB

func initDatabase() {
	// DigitalOcean injects the database connection string here
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Println("DATABASE_URL not found, skipping DB setup.")
		return
	}

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Error opening database connection: %v\n", err)
		return
	}

	// Create the tracking table automatically if it doesn't exist yet
	query := `
	CREATE TABLE IF NOT EXISTS speed_test_results (
		id SERIAL PRIMARY KEY,
		test_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		ping_ms REAL,
		jitter_ms REAL,
		download_mbps REAL,
		upload_mbps REAL
	);`

	_, err = db.Exec(query)
	if err != nil {
		log.Printf("Error creating speed_test_results table: %v\n", err)
		return
	}
	log.Println("Successfully connected to DigitalOcean PostgreSQL database!")
}

func main() {
	// Initialize the database before starting the server
	initDatabase()

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
		_, _ = io.Copy(io.Discard, r.Body)
		fmt.Fprintf(w, "ok")
	})

	// Serve frontend LAST
	fs := http.FileServer(http.Dir("frontend"))
	http.Handle("/", fs)

	fmt.Println("Server running on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error:", err)
	}
}
