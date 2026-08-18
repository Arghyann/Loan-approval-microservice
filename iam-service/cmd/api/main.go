package main

import (
	"fmt"
	"log"
	"net/http"

	"iam-service/internal/handlers"
)

func main() {
	// A simple test route
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "IAM Service is healthy!")
	})

	// Our new Registration Route!
	http.HandleFunc("/api/register", handlers.RegisterHandler)

	fmt.Println("Starting IAM Service on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}
