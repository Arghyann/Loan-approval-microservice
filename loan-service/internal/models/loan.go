package models

import (
	"time"
)

type LoanApplication struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	Amount           float64   `json:"amount"`
	Purpose          string    `json:"purpose"`
	TenureMonths     int       `json:"tenure_months"`
	MonthlyIncome    float64   `json:"monthly_income"`
	EmploymentType   string    `json:"employment_type"`
	Status           string    `json:"status"` // e.g., DRAFT, UNDER_REVIEW, APPROVED, REJECTED
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ApplyLoanRequest struct {
	Amount         float64 `json:"amount"`
	Purpose        string  `json:"purpose"`
	TenureMonths   int     `json:"tenure_months"`
	MonthlyIncome  float64 `json:"monthly_income"`
	EmploymentType string  `json:"employment_type"`
}
