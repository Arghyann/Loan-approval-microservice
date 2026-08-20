package auth

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateTokens creates a short-lived JWT access token and a long-lived refresh token
func GenerateTokens(userID string, role string) (string, string, time.Time, error) {
	// Read the private key
	keyBytes, err := os.ReadFile("private.pem")
	if err != nil {
		return "", "", time.Time{}, err
	}

	// Parse the private key
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
	if err != nil {
		return "", "", time.Time{}, err
	}

	// Create the Access Token claims (expires in 15 minutes)
	accessClaims := jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"exp":  time.Now().Add(15 * time.Minute).Unix(),
		"iat":  time.Now().Unix(),
	}

	// Sign the Access Token using RS256
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	accessToken, err := token.SignedString(privateKey)
	if err != nil {
		return "", "", time.Time{}, err
	}

	// Generate a random secure string for the Refresh Token
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", time.Time{}, err
	}
	refreshToken := hex.EncodeToString(bytes)

	// Refresh token expires in 7 days
	refreshExpiry := time.Now().Add(7 * 24 * time.Hour)

	return accessToken, refreshToken, refreshExpiry, nil
}
