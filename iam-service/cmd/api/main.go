package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"iam-service/internal/database"
	"iam-service/internal/handlers"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// 1. Load the .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	// 2. Read the connection string from the .env file
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL environment variable is not set!")
	}

	// 3. Connect to PostgreSQL
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Cannot connect to database: %v", err)
	}
	defer db.Close()

	// 4. Initialize the Repository and the Handler
	userRepo := database.NewUserRepository(db)
	authHandler := &handlers.AuthHandler{Repo: userRepo}

	// 5. Register the Routes
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "IAM Service is healthy!")
	})

	http.HandleFunc("/api/register", authHandler.RegisterHandler)
	http.HandleFunc("/api/login", authHandler.LoginHandler)

	fmt.Println("Starting IAM Service on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}

}
