package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"loan-service/internal/database"
	"loan-service/internal/models"
	"loan-service/internal/storage"

	"github.com/google/uuid"
)

type LoanHandler struct {
	Repo *database.LoanRepository
}

// ApplyLoanHandler receives the loan application form and saves it as DRAFT
func (h *LoanHandler) ApplyLoanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.ApplyLoanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Extract the actual UserID that our Auth Middleware injected into the context!
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
		log.Printf("DB Error: %v\n", err)
		http.Error(w, "Failed to save application", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(loan)
}

// DocumentUploadHandler generates a Presigned URL for Azure Blob Storage
func (h *LoanHandler) DocumentUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DocumentType string `json:"document_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("user_id").(string)
	fileName := fmt.Sprintf("%s_%s.pdf", userID, req.DocumentType)

	uploadURL, err := storage.GenerateUploadURL(fileName)
	if err != nil {
		log.Printf("Azure Storage Error: %v\n", err)
		http.Error(w, "Failed to generate upload URL", http.StatusInternalServerError)
		return
	}

	// Track this in PostgreSQL before returning it to the user
	storageURL := fmt.Sprintf("https://YOUR_ACCOUNT.blob.core.windows.net/kyc-documents/%s", fileName)
	h.Repo.RecordDocument(uuid.New().String(), userID, req.DocumentType, storageURL)

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
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DocumentType string `json:"document_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("user_id").(string)

	// Update PostgreSQL
	if err := h.Repo.MarkDocumentUploaded(userID, req.DocumentType); err != nil {
		log.Printf("DB Error: %v\n", err)
		http.Error(w, "Failed to update database", http.StatusInternalServerError)
		return
	}

	// =========================================================================
	// TODO: RABBITMQ INTEGRATION
	// The application is complete. Drop a message into RabbitMQ so the Python 
	// Risk Assessment Service knows it is time to run the ML model!
	// Example: publishEvent("EVALUATE_LOAN", loanID)
	// =========================================================================

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Document status updated to UPLOADED successfully"}`))
}
