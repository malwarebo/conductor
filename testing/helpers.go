package testing

import (
	"context"
	"time"

	"github.com/malwarebo/gopay/models"
)

func MockChargeRequest() *models.ChargeRequest {
	return &models.ChargeRequest{
		CustomerID:    "cus_test123",
		Amount:        1000,
		Currency:      "USD",
		PaymentMethod: "pm_test123",
		Description:   "Test payment",
		Metadata: map[string]interface{}{
			"order_id": "order_123",
		},
	}
}

func MockChargeResponse() *models.ChargeResponse {
	return &models.ChargeResponse{
		ID:               "ch_test123",
		CustomerID:       "cus_test123",
		Amount:           1000,
		Currency:         "USD",
		Status:           models.PaymentStatusSuccess,
		PaymentMethod:    "pm_test123",
		Description:      "Test payment",
		ProviderName:     "stripe",
		ProviderChargeID: "pi_test123",
		Metadata: map[string]interface{}{
			"order_id": "order_123",
		},
		CreatedAt: time.Now(),
	}
}

func MockRefundRequest() *models.RefundRequest {
	return &models.RefundRequest{
		PaymentID: "ch_test123",
		Amount:    1000,
		Reason:    "requested_by_customer",
		Metadata: map[string]interface{}{
			"refund_reason": "customer request",
		},
	}
}

func MockPayment() *models.Payment {
	return &models.Payment{
		ID:               "pay_test123",
		CustomerID:       "cus_test123",
		Amount:           1000,
		Currency:         "USD",
		Status:           models.PaymentStatusSuccess,
		PaymentMethod:    "pm_test123",
		Description:      "Test payment",
		ProviderName:     "stripe",
		ProviderChargeID: "pi_test123",
		Metadata: models.JSON{
			"order_id": "order_123",
		},
	}
}

func MockContext() context.Context {
	return context.Background()
}

func MockContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}
