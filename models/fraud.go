package models

import (
	"time"
)

type FraudAnalysisRequest struct {
	TransactionID       string  `json:"transaction_id"`
	UserID              string  `json:"user_id"`
	TransactionAmount   float64 `json:"transaction_amount"`
	BillingCountry      string  `json:"billing_country"`
	ShippingCountry     string  `json:"shipping_country"`
	IPAddress           string  `json:"ip_address"`
	TransactionVelocity int     `json:"transaction_velocity"`
}

type FraudAnalysisResponse struct {
	Allow  bool   `json:"allow"`
	Reason string `json:"reason,omitempty"`
}

type OpenAIFraudAssessment struct {
	IsFraudulent bool   `json:"is_fraudulent"`
	FraudScore   int    `json:"fraud_score"`
	Reason       string `json:"reason"`
}

type FraudAnalysisResult struct {
	ID                  string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TransactionID       string    `json:"transaction_id" gorm:"not null;index"`
	UserID              string    `json:"user_id" gorm:"not null;index"`
	TransactionAmount   float64   `json:"transaction_amount" gorm:"not null"`
	BillingCountry      string    `json:"billing_country" gorm:"not null"`
	ShippingCountry     string    `json:"shipping_country" gorm:"not null"`
	IPAddress           string    `json:"ip_address" gorm:"not null"`
	TransactionVelocity int       `json:"transaction_velocity" gorm:"not null"`
	IsFraudulent        bool      `json:"is_fraudulent" gorm:"not null"`
	FraudScore          int       `json:"fraud_score" gorm:"not null"`
	Reason              string    `json:"reason" gorm:"not null"`
	Allow               bool      `json:"allow" gorm:"not null"`
	CreatedAt           time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt           time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type FraudStatsRequest struct {
	StartDate string `json:"start_date" form:"start_date"`
	EndDate   string `json:"end_date" form:"end_date"`
}

type FraudStatsResponse struct {
	TotalTransactions               int     `json:"total_transactions"`
	TotalFraudulentTransactions     int     `json:"total_fraudulent_transactions"`
	AverageFraudScore               float64 `json:"average_fraud_score"`
	FraudulentTransactionPercentage float64 `json:"fraudulent_transaction_percentage"`
}
