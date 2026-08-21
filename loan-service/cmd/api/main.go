package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"loan-service/internal/database"
	"loan-service/internal/handlers"
	"loan-service/internal/middleware"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// 1. Load the IAM Public Key for JWT Verification
	if err := middleware.LoadPublicKey(); err != nil {
		log.Fatalf("Could not load public key: %v", err)
	}

	// 2. Connect to PostgreSQL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://admin:adminpassword@localhost:5432/microservice_db?sslmode=disable"
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}
	defer db.Close()

	// 3. Initialize the Repository and Handlers
	repo := &database.LoanRepository{DB: db}
	loanHandler := &handlers.LoanHandler{Repo: repo}

	// 4. Register Routes (Wrapped in RequireAuth middleware!)
	http.HandleFunc("/api/loans", middleware.RequireAuth(loanHandler.ApplyLoanHandler)) // For POST
	http.HandleFunc("/api/loans/", middleware.RequireAuth(loanHandler.GetLoansHandler)) // For GETs
	http.HandleFunc("/api/documents/upload-url", middleware.RequireAuth(loanHandler.DocumentUploadHandler))
	http.HandleFunc("/api/documents/confirm", middleware.RequireAuth(loanHandler.DocumentConfirmHandler))

	fmt.Println("Starting Loan Service on port 8081...")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}
