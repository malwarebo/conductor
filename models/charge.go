package models

import (
	"time"
)

type Charge struct {
	ID string `json:"id"`
	Amount int64 `json:"amount"`
	Currency string `json:"currency"`
	CustomerEmail string `json:"customer_email"`
	PaymentIntentId string `json:"payment_intent_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Status string `json:"status"`
}
