package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/segmentio/kafka-go"
)

type Message struct {
	TeamID    int64  `json:"team_id"`
	ChannelID int64  `json:"channel_id"`
	Content   string `json:"content"`
}

var kafkaWriter *kafka.Writer

func main() {
	// Kafka writer
	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP("host.docker.internal:9092"),
		Topic:    "team-messages",
		Balancer: &kafka.LeastBytes{},
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Post("/messages", handleMessage)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	log.Println("Messaging Service running on :8082")
	http.ListenAndServe(":8082", r)
}

func handleMessage(w http.ResponseWriter, r *http.Request) {
	var msg Message

	// Decode JSON body (team_id, channel_id, content)
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if msg.TeamID == 0 || msg.ChannelID == 0 || msg.Content == "" {
		http.Error(w, "team_id, channel_id, and content are required", http.StatusBadRequest)
		return
	}

	// Prepare Kafka message
	value, _ := json.Marshal(map[string]interface{}{
		"team_id":    msg.TeamID,
		"channel_id": msg.ChannelID,
		"content":    msg.Content,
	})

	err := kafkaWriter.WriteMessages(r.Context(), kafka.Message{
		Value: value,
	})

	if err != nil {
		log.Println("Kafka write error:", err)
		http.Error(w, "failed to publish message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Message published"))
}
