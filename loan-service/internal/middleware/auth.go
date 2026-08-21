package middleware

import (
	"context"
	"crypto/rsa"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

var publicKey *rsa.PublicKey

// LoadPublicKey reads the public.pem file when the server starts
func LoadPublicKey() error {
	keyData, err := os.ReadFile("public.pem")
	if err != nil {
		return err
	}
	key, err := jwt.ParseRSAPublicKeyFromPEM(keyData)
	if err != nil {
		return err
	}
	publicKey = key
	return nil
}

// RequireAuth intercepts the HTTP request and verifies the JWT token
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Verify the token using the Public Key (No need to talk to IAM service!)
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return publicKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Extract the user_id from the token payload
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		userID := claims["sub"].(string) // "sub" is the standard JWT field for User ID

		// Inject the userID into the Request Context so the handlers can use it
		ctx := context.WithValue(r.Context(), "user_id", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
