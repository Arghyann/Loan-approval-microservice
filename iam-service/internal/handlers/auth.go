package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"iam-service/internal/auth"
	"iam-service/internal/database"
	"iam-service/internal/models"
	
	"github.com/google/uuid"
)

// 1. We create a struct to hold our Database Repository!
type AuthHandler struct {
	Repo *database.UserRepository
}

// 2. Notice this is now `func (h *AuthHandler) RegisterHandler...`
func (h *AuthHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	user := models.User{
		ID:           uuid.New().String(),
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         "customer",
		CreatedAt:    time.Now(),
	}

	// 3. We use h.Repo.Create to save to the database!
	if err := h.Repo.Create(user); err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// 4. Send the final success response (without the double Fprintf error)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *AuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}
	user, err := h.Repo.GetUserByEmail(req.Email)
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}
	
	if err := auth.CheckPassword(user.PasswordHash, req.Password); err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}
	authToken, refreshToken, refreshExpiry,err := auth.GenerateTokens(user.ID, user.Role)
	if err != nil {
		http.Error(w, "Failed to generate tokens", http.StatusInternalServerError)
		return
	}
	if err := h.Repo.SaveRefreshToken(refreshToken, user.ID, refreshExpiry); err != nil {
		http.Error(w, "Failed to save refresh token", http.StatusInternalServerError)
		return
	}
	
	response := map[string]string{
		"auth_token":    authToken,
		"refresh_token": refreshToken,
		"expires_at":    refreshExpiry.Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)


	
}
