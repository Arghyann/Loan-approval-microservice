package database

import (
	"database/sql"
	"loan-service/internal/models"
)

type LoanRepository struct {
	DB *sql.DB
}

// SaveApplication inserts the new DRAFT loan into the database
func (repo *LoanRepository) SaveApplication(loan models.LoanApplication) error {
	query := `
		INSERT INTO loans (id, user_id, amount, purpose, tenure_months, monthly_income, employment_type, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := repo.DB.Exec(query,
		loan.ID, loan.UserID, loan.Amount, loan.Purpose, loan.TenureMonths,
		loan.MonthlyIncome, loan.EmploymentType, loan.Status, loan.CreatedAt, loan.UpdatedAt,
	)
	return err
}

// RecordDocument tracks a new document upload link in the database
func (repo *LoanRepository) RecordDocument(docID, userID, docType, storageURL string) error {
	query := `
		INSERT INTO documents (id, user_id, document_type, storage_url, status)
		VALUES ($1, $2, $3, $4, 'PENDING')
	`
	_, err := repo.DB.Exec(query, docID, userID, docType, storageURL)
	return err
}

// MarkDocumentUploaded updates the status when Azure confirms the upload
func (repo *LoanRepository) MarkDocumentUploaded(userID, docType string) error {
	query := `
		UPDATE documents 
		SET status = 'UPLOADED' 
		WHERE user_id = $1 AND document_type = $2
	`
	_, err := repo.DB.Exec(query, userID, docType)
	return err
}
