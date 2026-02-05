package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"
)

type Message struct {
	ID        int64  `json:"id"`
	TeamID    int64  `json:"team_id"`
	ChannelID int64  `json:"channel_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

var db *sql.DB

func main() {
	// Connect to PostgreSQL (your collab DB)
	connStr := "postgres://postgres:YOUR_PASSWORD@localhost:5432/collab?sslmode=disable"

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("DB connection error:", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal("DB unreachable:", err)
	}

	// Start Kafka consumer
	go startKafkaConsumer()

	// REST API
	r := chi.NewRouter()
	r.Get("/teams/{teamID}/messages", getMessages)
	r.Get("/teams/{teamID}/channels/{channelID}/messages", getChannelMessages)

	log.Println("History Service running on :8084")
	http.ListenAndServe(":8084", r)
}

func startKafkaConsumer() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{"host.docker.internal:9092"},
		Topic:    "team-messages",
		GroupID:  "history-consumers",
		MinBytes: 1,
		MaxBytes: 10e6,
		MaxWait:  1 * time.Second,
	})

	log.Println("History Kafka consumer started...")

	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Println("Kafka read error:", err)
			continue
		}

		// Decode incoming JSON
		var incoming struct {
			TeamID    int64  `json:"team_id"`
			ChannelID int64  `json:"channel_id"`
			Content   string `json:"content"`
		}

		if err := json.Unmarshal(msg.Value, &incoming); err != nil {
			log.Println("JSON decode error:", err)
			continue
		}

		// Insert into DB
		_, err = db.Exec(`
            INSERT INTO messages (team_id, channel_id, content)
            VALUES ($1, $2, $3)
        `, incoming.TeamID, incoming.ChannelID, incoming.Content)

		if err != nil {
			log.Println("DB insert error:", err)
		}
	}
}

func getMessages(w http.ResponseWriter, r *http.Request) {
	teamIDStr := chi.URLParam(r, "teamID")
	teamID, _ := strconv.ParseInt(teamIDStr, 10, 64)

	rows, err := db.Query(`
        SELECT id, team_id, channel_id, content, created_at
        FROM messages
        WHERE team_id = $1
        ORDER BY created_at DESC
        LIMIT 50
    `, teamID)

	if err != nil {
		http.Error(w, "query error", 500)
		return
	}
	defer rows.Close()

	var messages []Message

	for rows.Next() {
		var m Message
		rows.Scan(&m.ID, &m.TeamID, &m.ChannelID, &m.Content, &m.CreatedAt)
		messages = append(messages, m)
	}

	json.NewEncoder(w).Encode(messages)
}

func getChannelMessages(w http.ResponseWriter, r *http.Request) {
	teamID, _ := strconv.ParseInt(chi.URLParam(r, "teamID"), 10, 64)
	channelID, _ := strconv.ParseInt(chi.URLParam(r, "channelID"), 10, 64)

	rows, err := db.Query(`
        SELECT id, team_id, channel_id, content, created_at
        FROM messages
        WHERE team_id = $1 AND channel_id = $2
        ORDER BY created_at DESC
        LIMIT 50
    `, teamID, channelID)

	if err != nil {
		http.Error(w, "query error", 500)
		return
	}
	defer rows.Close()

	var messages []Message

	for rows.Next() {
		var m Message
		rows.Scan(&m.ID, &m.TeamID, &m.ChannelID, &m.Content, &m.CreatedAt)
		messages = append(messages, m)
	}

	json.NewEncoder(w).Encode(messages)
}
