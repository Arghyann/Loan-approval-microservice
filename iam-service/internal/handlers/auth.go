package handlers

import (
	"encoding/json"
	"fmt"
	"time"
	"net/http"
    "golang.org/x/crypto/bcrypt"
	"iam-service/internal/models"
	"github.com/google/uuid"
)

func hashPassword(password string ) (string, error){
	    bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)                                                                       
        return string(bytes), err
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Ensure this is a POST request
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 2. Read the JSON from the request body into our struct
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == ""{
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return 
	}

	hashedPassword, err	:= hashPassword(req.Password)
	if err!=nil{
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
        return
	}
	user := models.User{
            ID:           uuid.New().String(), // <-- Go generates the unique ID instantly!
            Email:        req.Email,
            PasswordHash: hashedPassword,
            Role:         "customer",
            CreatedAt:    time.Now(),
        }

	// Save user to database
    if err := userRepository.Create(user); err != nil {
        http.Error(w, "Failed to create user", http.StatusInternalServerError)
        return
    }

    // Return response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)

    json.NewEncoder(w).Encode(user)
	// 4. Send a success response
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Success! We received the registration for: %s\n", req.Email)
}
