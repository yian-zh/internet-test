package main

import (
    "fmt"
    "net/http"
    "time"
)

func main() {
    http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/plain")
        fmt.Fprintf(w, "%d", time.Now().UnixMilli())
    })

    http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
        data := make([]byte, 10*1024*1024) // 10 MB
        w.Header().Set("Content-Type", "application/octet-stream")
        w.Write(data)
    })

    http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
        r.Body.Close()
        fmt.Fprintf(w, "ok")
    })

    fmt.Println("Server running on :8080")
    http.ListenAndServe(":8080", nil)
}