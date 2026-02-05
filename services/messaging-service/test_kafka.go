package main

import (
	"fmt"

	"github.com/segmentio/kafka-go"
)

func main() {
	fmt.Println("Trying to connect to Kafka on localhost:9092...")

	conn, err := kafka.Dial("tcp", "localhost:9092")
	if err != nil {
		fmt.Println("FAILED:", err)
		return
	}

	fmt.Println("SUCCESS: Connected to Kafka!")
	conn.Close()
}
