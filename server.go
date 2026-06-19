package main

import (
	"database/sql"
	"encoding/json" // Added to handle incoming speed test JSON data
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

var db *sql.DB

// TestResult represents the structure of incoming speed test metrics for saving
type TestResult struct {
	PingMS          float64 `json:"ping_ms"`
	JitterMS        float64 `json:"jitter_ms"`
	DownloadMbps    float64 `json:"download_mbps"`
	UploadMbps      float64 `json:"upload_mbps"`
	TestingPlatform string  `json:"testing_platform"` // Identifies the platform/server used
}

// DBTestResult represents the structure used to retrieve data including the timestamp
type DBTestResult struct {
	TestTime        time.Time `json:"test_time"`
	PingMS          float64   `json:"ping_ms"`
	JitterMS        float64   `json:"jitter_ms"`
	DownloadMbps    float64   `json:"download_mbps"`
	UploadMbps      float64   `json:"upload_mbps"`
	TestingPlatform string    `json:"testing_platform"`
}

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

	// Create the tracking table automatically with testing_platform column included
	query := `
	CREATE TABLE IF NOT EXISTS speed_test_results (
		id SERIAL PRIMARY KEY,
		test_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		ping_ms REAL,
		jitter_ms REAL,
		download_mbps REAL,
		upload_mbps REAL,
		testing_platform VARCHAR(50) DEFAULT 'LibreSpeed-DO'
	);`

	_, err = db.Exec(query)
	if err != nil {
		log.Printf("Error creating speed_test_results table: %v\n", err)
		return
	}

	// INFRASTRUCTURE RUNTIME MIGRATION PATCH:
	// Forcefully append the testing_platform column to old pre-existing tables to prevent 500 runtime execution drop outs.
	_, _ = db.Exec(`ALTER TABLE speed_test_results ADD COLUMN IF NOT EXISTS testing_platform VARCHAR(50) DEFAULT 'LibreSpeed-DO';`)

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

	// Save & Fetch Results endpoint - Supports writing telemetry and reading log arrays
	http.HandleFunc("/save-results", func(w http.ResponseWriter, r *http.Request) {
		// Allow CORS requests from frontend
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight CORS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Double check if database connection exists
		if db == nil {
			http.Error(w, "Database connection not active", http.StatusInternalServerError)
			return
		}

		// --- CASE 1: FETCH DATA (GET METHOD) ---
		if r.Method == http.MethodGet {
			// Query the last 15 test rows from the cluster ordered by newest first
			query := `SELECT test_time, ping_ms, jitter_ms, download_mbps, upload_mbps, testing_platform FROM speed_test_results ORDER BY test_time DESC LIMIT 15`
			rows, err := db.Query(query)
			if err != nil {
				log.Printf("Error fetching rows from database: %v\n", err)
				http.Error(w, "Failed to read logs", http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			resultsArray := []DBTestResult{}
			for rows.Next() {
				var res DBTestResult
				err := rows.Scan(&res.TestTime, &res.PingMS, &res.JitterMS, &res.DownloadMbps, &res.UploadMbps, &res.TestingPlatform)
				if err != nil {
					log.Printf("Error mapping result row: %v\n", err)
					continue
				}
				resultsArray = append(resultsArray, res)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resultsArray)
			return
		}

		// --- CASE 2: SAVE DATA (POST METHOD) ---
		if r.Method == http.MethodPost {
			var data TestResult
			err := json.NewDecoder(r.Body).Decode(&data)
			if err != nil {
				log.Printf("Error decoding JSON payload: %v\n", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Insert metrics along with the platform identity row
			query := `INSERT INTO speed_test_results (ping_ms, jitter_ms, download_mbps, upload_mbps, testing_platform) VALUES ($1, $2, $3, $4, $5)`
			_, err = db.Exec(query, data.PingMS, data.JitterMS, data.DownloadMbps, data.UploadMbps, data.TestingPlatform)
			if err != nil {
				log.Printf("Database insertion crash: %v\n", err)
				http.Error(w, "Failed to write data to database", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Success")
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
