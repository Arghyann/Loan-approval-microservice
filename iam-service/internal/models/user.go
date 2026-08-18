package models

import "time"

// User represents a customer in the database
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // The '-' means this field is NEVER sent back in JSON responses
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

// RegisterRequest represents the JSON we expect from the Frontend
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
