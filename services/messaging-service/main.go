package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/segmentio/kafka-go"
)

var kafkaWriter *kafka.Writer

type IncomingMessage struct {
	TeamID  int64  `json:"team_id"`
	Content string `json:"content"`
}

func main() {
	// IMPORTANT: Kafka inside Docker Desktop must be reached via host.docker.internal
	broker := "host.docker.internal:9092"

	// Quick connectivity check
	conn, err := net.DialTimeout("tcp", broker, 5*time.Second)
	if err != nil {
		log.Fatalf("Kafka broker unreachable at %s: %v", broker, err)
	}
	conn.Close()

	// Kafka writer
	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(broker),
		Topic:    "team-messages",
		Balancer: &kafka.LeastBytes{},
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/messages", handleIncomingMessage)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		c, err := net.DialTimeout("tcp", broker, 2*time.Second)
		if err != nil {
			http.Error(w, fmt.Sprintf("Kafka unreachable: %v", err), http.StatusServiceUnavailable)
			return
		}
		c.Close()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:    ":8082",
		Handler: r,
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		log.Println("Shutting down messaging service...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if kafkaWriter != nil {
			kafkaWriter.Close()
		}
		srv.Shutdown(ctx)
	}()

	log.Println("Messaging Service running on :8082")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("ListenAndServe: %v", err)
	}
}

func handleIncomingMessage(w http.ResponseWriter, r *http.Request) {
	var msg IncomingMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if msg.TeamID == 0 || msg.Content == "" {
		http.Error(w, "team_id and content required", http.StatusBadRequest)
		return
	}

	payload, _ := json.Marshal(msg)

	err := kafkaWriter.WriteMessages(
		context.Background(),
		kafka.Message{
			Key:   []byte(time.Now().Format(time.RFC3339Nano)),
			Value: payload,
		},
	)

	if err != nil {
		log.Printf("FAILED TO WRITE MESSAGE: %+v\n", err)
		http.Error(w, "failed to publish message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Message published"))
}
