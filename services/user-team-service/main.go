package main

import (
    "database/sql"
    "log"
    "net/http"

    _ "github.com/lib/pq"
)

func main() {
    // Connect to your local Postgres 18
    dbURL := "postgres://app:app@localhost:5432/collab?sslmode=disable"

    db, err := sql.Open("postgres", dbURL)
    if err != nil {
        log.Fatalf("failed to open db: %v", err)
    }

    if err := db.Ping(); err != nil {
        log.Fatalf("failed to connect to db: %v", err)
    }

    log.Println("Connected to Postgres successfully!")

    // Temporary test endpoint
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("User & Team Service is running"))
    })

    log.Println("Server running on :8081")
    log.Fatal(http.ListenAndServe(":8081", nil))
}
