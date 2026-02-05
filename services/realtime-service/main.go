package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Client struct {
	conn *websocket.Conn
	send chan []byte
}

var clients = make(map[*Client]bool)
var broadcast = make(chan []byte)

func main() {
	go startKafkaConsumer()
	go handleBroadcasts()

	http.HandleFunc("/ws", handleWebSocket)

	log.Println("Realtime Service running on :8083")
	if err := http.ListenAndServe(":8083", nil); err != nil {
		log.Fatal(err)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	client := &Client{
		conn: conn,
		send: make(chan []byte),
	}

	clients[client] = true

	go writePump(client)
}

func writePump(c *Client) {
	for msg := range c.send {
		err := c.conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println("WebSocket write error:", err)
			c.conn.Close()
			delete(clients, c)
			return
		}
	}
}

func handleBroadcasts() {
	for msg := range broadcast {
		for client := range clients {
			select {
			case client.send <- msg:
			default:
				close(client.send)
				delete(clients, client)
			}
		}
	}
}

func startKafkaConsumer() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{"host.docker.internal:9092"},
		Topic:    "team-messages",
		GroupID:  "realtime-consumers",
		MinBytes: 1,
		MaxBytes: 10e6,
		MaxWait:  1 * time.Second,
	})

	log.Println("Kafka consumer started...")

	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Println("Kafka read error:", err)
			continue
		}

		// msg.Value already contains team_id, channel_id, content
		fmt.Printf("Received message: %s\n", string(msg.Value))

		// Broadcast the full JSON to all WebSocket clients
		broadcast <- msg.Value
	}
}
