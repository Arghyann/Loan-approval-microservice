package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"iam-service/internal/auth"
	"iam-service/internal/database"
	"iam-service/internal/models"

	"github.com/google/uuid"
)

type AuthHandler struct {
	Repo *database.UserRepository
}

func (h *AuthHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodPost {
		slog.Warn("method not allowed", "endpoint", "/api/register", "method", r.Method, "ip", r.RemoteAddr)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("invalid request body", "endpoint", "/api/register", "ip", r.RemoteAddr, "error", err.Error())
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		slog.Warn("missing required fields", "endpoint", "/api/register", "ip", r.RemoteAddr)
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		slog.Error("failed to hash password", "endpoint", "/api/register", "email", req.Email, "error", err.Error())
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

	if err := h.Repo.Create(user); err != nil {
		slog.Error("failed to create user", "endpoint", "/api/register", "email", req.Email, "error", err.Error())
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	slog.Info("user registered",
		"endpoint", "/api/register",
		"user_id", user.ID,
		"email", user.Email,
		"ip", r.RemoteAddr,
		"latency_ms", time.Since(start).Milliseconds(),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *AuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodPost {
		slog.Warn("method not allowed", "endpoint", "/api/login", "method", r.Method, "ip", r.RemoteAddr)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("invalid request body", "endpoint", "/api/login", "ip", r.RemoteAddr, "error", err.Error())
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		slog.Warn("missing required fields", "endpoint", "/api/login", "ip", r.RemoteAddr)
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	user, err := h.Repo.GetUserByEmail(req.Email)
	if err != nil {
		slog.Warn("login failed: user not found", "endpoint", "/api/login", "email", req.Email, "ip", r.RemoteAddr)
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	if err := auth.CheckPassword(user.PasswordHash, req.Password); err != nil {
		slog.Warn("login failed: invalid password",
			"endpoint", "/api/login",
			"email", req.Email,
			"user_id", user.ID,
			"ip", r.RemoteAddr,
		)
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	authToken, refreshToken, refreshExpiry, err := auth.GenerateTokens(user.ID, user.Role)
	if err != nil {
		slog.Error("failed to generate tokens", "endpoint", "/api/login", "user_id", user.ID, "error", err.Error())
		http.Error(w, "Failed to generate tokens", http.StatusInternalServerError)
		return
	}

	if err := h.Repo.SaveRefreshToken(refreshToken, user.ID, refreshExpiry); err != nil {
		slog.Error("failed to save refresh token", "endpoint", "/api/login", "user_id", user.ID, "error", err.Error())
		http.Error(w, "Failed to save refresh token", http.StatusInternalServerError)
		return
	}

	slog.Info("user logged in",
		"endpoint", "/api/login",
		"user_id", user.ID,
		"email", user.Email,
		"ip", r.RemoteAddr,
		"latency_ms", time.Since(start).Milliseconds(),
	)

	response := map[string]string{
		"auth_token":    authToken,
		"refresh_token": refreshToken,
		"expires_at":    refreshExpiry.Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) RefreshHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodPost {
		slog.Warn("method not allowed", "endpoint", "/api/refresh", "method", r.Method, "ip", r.RemoteAddr)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		slog.Warn("invalid request body", "endpoint", "/api/refresh", "ip", r.RemoteAddr)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID, err := h.Repo.GetUserIDFromRefreshToken(req.RefreshToken)
	if err != nil {
		slog.Warn("invalid or expired refresh token", "endpoint", "/api/refresh", "ip", r.RemoteAddr)
		http.Error(w, "Invalid or expired refresh token", http.StatusUnauthorized)
		return
	}

	user, err := h.Repo.GetUserByID(userID)
	if err != nil {
		slog.Error("user not found after valid refresh token", "endpoint", "/api/refresh", "user_id", userID, "error", err.Error())
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	authToken, newRefreshToken, refreshExpiry, err := auth.GenerateTokens(user.ID, user.Role)
	if err != nil {
		slog.Error("failed to generate tokens", "endpoint", "/api/refresh", "user_id", user.ID, "error", err.Error())
		http.Error(w, "Failed to generate tokens", http.StatusInternalServerError)
		return
	}

	h.Repo.DeleteRefreshToken(req.RefreshToken)
	h.Repo.SaveRefreshToken(newRefreshToken, user.ID, refreshExpiry)

	slog.Info("tokens refreshed",
		"endpoint", "/api/refresh",
		"user_id", user.ID,
		"ip", r.RemoteAddr,
		"latency_ms", time.Since(start).Milliseconds(),
	)

	response := map[string]string{
		"auth_token":    authToken,
		"refresh_token": newRefreshToken,
		"expires_at":    refreshExpiry.Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodPost {
		slog.Warn("method not allowed", "endpoint", "/api/change-password", "method", r.Method, "ip", r.RemoteAddr)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("invalid request body", "endpoint", "/api/change-password", "ip", r.RemoteAddr, "error", err.Error())
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.Repo.GetUserByEmail(req.Email)
	if err != nil {
		slog.Warn("change password failed: user not found", "endpoint", "/api/change-password", "email", req.Email, "ip", r.RemoteAddr)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := auth.CheckPassword(user.PasswordHash, req.OldPassword); err != nil {
		slog.Warn("change password failed: invalid old password",
			"endpoint", "/api/change-password",
			"user_id", user.ID,
			"ip", r.RemoteAddr,
		)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		slog.Error("failed to hash new password", "endpoint", "/api/change-password", "user_id", user.ID, "error", err.Error())
		http.Error(w, "Failed to process new password", http.StatusInternalServerError)
		return
	}

	if err := h.Repo.UpdatePassword(user.Email, newHash); err != nil {
		slog.Error("failed to update password in db", "endpoint", "/api/change-password", "user_id", user.ID, "error", err.Error())
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	slog.Info("password changed",
		"endpoint", "/api/change-password",
		"user_id", user.ID,
		"ip", r.RemoteAddr,
		"latency_ms", time.Since(start).Milliseconds(),
	)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Password updated successfully"}`))
}
