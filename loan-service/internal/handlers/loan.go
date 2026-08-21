package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"loan-service/internal/database"
	"loan-service/internal/models"
	"loan-service/internal/storage"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/google/uuid"
)

type LoanHandler struct {
	Repo *database.LoanRepository
}

// ApplyLoanHandler receives the loan application form and saves it as DRAFT
func (h *LoanHandler) ApplyLoanHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodPost {
		slog.Warn("method not allowed", "endpoint", "/api/loans", "method", r.Method, "ip", r.RemoteAddr)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.ApplyLoanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("invalid request body", "endpoint", "/api/loans", "ip", r.RemoteAddr, "error", err.Error())
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("user_id").(string)

	loan := models.LoanApplication{
		ID:             uuid.New().String(),
		UserID:         userID,
		Amount:         req.Amount,
		Purpose:        req.Purpose,
		TenureMonths:   req.TenureMonths,
		MonthlyIncome:  req.MonthlyIncome,
		EmploymentType: req.EmploymentType,
		Status:         "DRAFT",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := h.Repo.SaveApplication(loan); err != nil {
		slog.Error("failed to save loan application", "endpoint", "/api/loans", "user_id", userID, "error", err.Error())
		http.Error(w, "Failed to save application", http.StatusInternalServerError)
		return
	}

	slog.Info("loan application created",
		"endpoint", "/api/loans",
		"user_id", userID,
		"loan_id", loan.ID,
		"amount", loan.Amount,
		"ip", r.RemoteAddr,
		"latency_ms", time.Since(start).Milliseconds(),
	)

	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		slog.Error("RABBITMQ_URL not set in environment variables")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if conn, err := amqp.Dial(rabbitmqURL); err == nil {                                                                                  
            defer conn.Close()                                                                                                                    
            if ch, err := conn.Channel(); err == nil {
                defer ch.Close()
  
                q, err := ch.QueueDeclare("loan_applications", true, false, false, false, nil)
                if err == nil {
                    body, _ := json.Marshal(loan)
                    err = ch.Publish(
                        "", q.Name, false, false,
                        amqp.Publishing{
                            ContentType: "application/json",
                            Body:        body,
                        },
                    )
                    if err != nil {
                        slog.Error("failed to publish to RabbitMQ", "error", err.Error())
                    } else {
                        slog.Info("published to RabbitMQ", "loan_id", loan.ID)
                    }
                }
            }
        } else {
            slog.Error("failed to connect to RabbitMQ", "error", err.Error())
        }
        // --- RABBITMQ END ---
  
	if conn, err := amqp.Dial(rabbitmqURL); err == nil {
		defer conn.Close()
		if ch, err := conn.Channel(); err == nil {
			defer ch.Close()

			q, err := ch.QueueDeclare("loan_applications", true, false, false, false, nil)
			if err == nil {
				body, _ := json.Marshal(loan)
				err = ch.Publish(
					"", q.Name, false, false,
					amqp.Publishing{
						ContentType: "application/json",
						Body:        body,
					},
				)
				if err != nil {
					slog.Error("failed to publish to RabbitMQ", "error", err.Error())
				} else {
					slog.Info("published to RabbitMQ", "loan_id", loan.ID)
				}
			}
		}
	} else {
		slog.Error("failed to connect to RabbitMQ", "error", err.Error())
	}
	// --- RABBITMQ END ---

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(loan)
}

// DocumentUploadHandler generates a Presigned URL for Azure Blob Storage
func (h *LoanHandler) DocumentUploadHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodPost {
		slog.Warn("method not allowed", "endpoint", "/api/documents/upload-url", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DocumentType string `json:"document_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("invalid request body", "endpoint", "/api/documents/upload-url", "error", err.Error())
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("user_id").(string)
	fileName := fmt.Sprintf("%s_%s.pdf", userID, req.DocumentType)

	uploadURL, err := storage.GenerateUploadURL(fileName)
	if err != nil {
		slog.Error("failed to generate azure presigned url", "endpoint", "/api/documents/upload-url", "user_id", userID, "error", err.Error())
		http.Error(w, "Failed to generate upload URL", http.StatusInternalServerError)
		return
	}

	storageURL := fmt.Sprintf("https://YOUR_ACCOUNT.blob.core.windows.net/kyc-documents/%s", fileName)
	if err := h.Repo.RecordDocument(uuid.New().String(), userID, req.DocumentType, storageURL); err != nil {
		slog.Error("failed to record document in db", "endpoint", "/api/documents/upload-url", "user_id", userID, "error", err.Error())
	}

	slog.Info("presigned url generated",
		"endpoint", "/api/documents/upload-url",
		"user_id", userID,
		"document_type", req.DocumentType,
		"latency_ms", time.Since(start).Milliseconds(),
	)

	response := map[string]string{
		"upload_url": uploadURL,
		"file_name":  fileName,
		"expires_in": "15 minutes",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DocumentConfirmHandler is called by the React frontend AFTER Azure returns 200 OK.
func (h *LoanHandler) DocumentConfirmHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodPost {
		slog.Warn("method not allowed", "endpoint", "/api/documents/confirm", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DocumentType string `json:"document_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("invalid request body", "endpoint", "/api/documents/confirm", "error", err.Error())
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("user_id").(string)

	if err := h.Repo.MarkDocumentUploaded(userID, req.DocumentType); err != nil {
		slog.Error("failed to update document status", "endpoint", "/api/documents/confirm", "user_id", userID, "error", err.Error())
		http.Error(w, "Failed to update database", http.StatusInternalServerError)
		return
	}

	slog.Info("document marked as uploaded",
		"endpoint", "/api/documents/confirm",
		"user_id", userID,
		"document_type", req.DocumentType,
		"latency_ms", time.Since(start).Milliseconds(),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Document status updated to UPLOADED successfully"}`))
}

// GetLoansHandler handles both listing all loans and getting a specific loan status
func (h *LoanHandler) GetLoansHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodGet {
		slog.Warn("method not allowed", "endpoint", "/api/loans/", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value("user_id").(string)
	path := r.URL.Path[len("/api/loans/"):]

	w.Header().Set("Content-Type", "application/json")

	if path == "" {
		loans, err := h.Repo.GetLoansByUserID(userID)
		if err != nil {
			slog.Error("failed to fetch user loans", "endpoint", "/api/loans/", "user_id", userID, "error", err.Error())
			http.Error(w, "Failed to fetch loans", http.StatusInternalServerError)
			return
		}
		if loans == nil {
			loans = []models.LoanApplication{}
		}
		
		slog.Info("fetched all user loans", "endpoint", "/api/loans/", "user_id", userID, "count", len(loans), "latency_ms", time.Since(start).Milliseconds())
		json.NewEncoder(w).Encode(loans)
		return
	}

	loan, err := h.Repo.GetLoanByID(path)
	if err != nil {
		slog.Warn("loan not found", "endpoint", "/api/loans/{id}", "loan_id", path, "user_id", userID)
		http.Error(w, "Loan not found", http.StatusNotFound)
		return
	}

	if loan.UserID != userID {
		slog.Warn("idor attempt detected", "endpoint", "/api/loans/{id}", "loan_id", path, "attempted_by_user", userID)
		http.Error(w, "Forbidden: You do not have access to this loan", http.StatusForbidden)
		return
	}

	slog.Info("fetched specific loan", "endpoint", "/api/loans/{id}", "user_id", userID, "loan_id", loan.ID, "latency_ms", time.Since(start).Milliseconds())
	json.NewEncoder(w).Encode(loan)
}
