package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/messages", handleIncomingMessage)

	log.Println("Messaging Service running on :8082")
	log.Fatal(http.ListenAndServe(":8082", r))
}

func handleIncomingMessage(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Message received"))
}
