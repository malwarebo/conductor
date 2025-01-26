package models

import (
	"time"
)

type PricingType string
type SubscriptionStatus string
type BillingPeriod string

const (
	PricingTypeFixed    PricingType = "fixed"
	PricingTypePerUnit  PricingType = "per_unit"
	PricingTypeTiered   PricingType = "tiered"
	PricingTypeVolume   PricingType = "volume"

	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusCanceled  SubscriptionStatus = "canceled"
	SubscriptionStatusPaused    SubscriptionStatus = "paused"
	SubscriptionStatusTrialing  SubscriptionStatus = "trialing"
	SubscriptionStatusPastDue   SubscriptionStatus = "past_due"

	BillingPeriodDaily    BillingPeriod = "daily"
	BillingPeriodWeekly   BillingPeriod = "weekly"
	BillingPeriodMonthly  BillingPeriod = "monthly"
	BillingPeriodYearly   BillingPeriod = "yearly"
)

type Plan struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	Amount        float64     `json:"amount"`
	Currency      string      `json:"currency"`
	BillingPeriod BillingPeriod `json:"billing_period"`
	PricingType   PricingType `json:"pricing_type"`
	TrialDays     int         `json:"trial_days"`
	Features      []string    `json:"features"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

type Subscription struct {
	ID              string             `json:"id"`
	CustomerID      string             `json:"customer_id"`
	PlanID          string             `json:"plan_id"`
	Status          SubscriptionStatus `json:"status"`
	CurrentPeriodStart time.Time       `json:"current_period_start"`
	CurrentPeriodEnd   time.Time       `json:"current_period_end"`
	CanceledAt      *time.Time         `json:"canceled_at,omitempty"`
	TrialStart      *time.Time         `json:"trial_start,omitempty"`
	TrialEnd        *time.Time         `json:"trial_end,omitempty"`
	Quantity        int                `json:"quantity"`
	PaymentMethodID string             `json:"payment_method_id"`
	ProviderName    string             `json:"provider_name"`
	Metadata        map[string]string  `json:"metadata,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

type CreateSubscriptionRequest struct {
	CustomerID      string             `json:"customer_id"`
	PlanID          string             `json:"plan_id"`
	PaymentMethodID string             `json:"payment_method_id"`
	Quantity        int                `json:"quantity"`
	TrialDays       *int               `json:"trial_days,omitempty"`
	Metadata        map[string]string  `json:"metadata,omitempty"`
}

type UpdateSubscriptionRequest struct {
	Quantity        *int               `json:"quantity,omitempty"`
	PlanID          *string            `json:"plan_id,omitempty"`
	PaymentMethodID *string            `json:"payment_method_id,omitempty"`
	Metadata        map[string]string  `json:"metadata,omitempty"`
}

type CancelSubscriptionRequest struct {
	CancelAtPeriodEnd bool               `json:"cancel_at_period_end"`
	Reason            string             `json:"reason,omitempty"`
}

type SubscriptionEvent struct {
	ID              string             `json:"id"`
	SubscriptionID  string             `json:"subscription_id"`
	Type            string             `json:"type"` // created, updated, canceled, payment_failed, etc.
	Data            interface{}        `json:"data"`
	CreatedAt       time.Time          `json:"created_at"`
}
